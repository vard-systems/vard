package oidc

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/store"
)

type AuthorizeParams struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	Audience            string
	Prompt              string
	MaxAge              string
	LoginHint           string
	ResponseMode        string
	// DPoPKey is dpop_jkt (RFC 9449 §10): the thumbprint of the DPoP key the
	// code may only be redeemed with. A pushed request also takes it from
	// the push's own proof.
	DPoPKey string
	// RequestURI names a request the client pushed first (RFC 9126); the
	// parameters are then the pushed ones, not the address's.
	RequestURI string `json:"-"`
	// Repeated names a parameter the request gave more than once.
	Repeated string `json:"-"`
}

const (
	maxState     = 1024
	maxNonce     = 256
	maxScope     = 1024
	maxLoginHint = 255
	maxMethod    = 8
	maxAudience  = 255
)

func (p AuthorizeParams) oversized() string {
	limits := []struct {
		name  string
		value string
		max   int
	}{
		{"state", p.State, maxState},
		{"nonce", p.Nonce, maxNonce},
		{"scope", p.Scope, maxScope},
		{"login_hint", p.LoginHint, maxLoginHint},
		{"code_challenge_method", p.CodeChallengeMethod, maxMethod},
		{"audience", p.Audience, maxAudience},
	}

	for _, limit := range limits {
		switch {
		case len(limit.value) > limit.max:
			return limit.name + " is longer than " + strconv.Itoa(limit.max) + " bytes"
		case !utf8.ValidString(limit.value) || strings.ContainsRune(limit.value, 0):
			return limit.name + " is not valid UTF-8"
		}
	}

	return ""
}

func (s *Service) Authorize(ctx context.Context, p AuthorizeParams, session *Session) (string, error) {
	if p.Repeated != "" {
		return s.errorPage(ErrInvalidRequest, p.Repeated+" is repeated"), nil
	}

	pushed := p.RequestURI != ""
	if pushed {
		taken, page, err := s.takePushed(ctx, p)
		if err != nil || page != "" {
			return page, err
		}
		p = *taken
	}

	app, err := s.store.ApplicationByClientID(ctx, p.ClientID)
	switch {
	case p.ClientID == "":
		return s.errorPage(ErrInvalidRequest, "client_id is required"), nil
	case errors.Is(err, store.ErrNotFound):
		return s.errorPage(ErrInvalidRequest, "no application has this client_id"), nil
	case err != nil:
		return "", err
	}

	if app.RequirePushedRequests && !pushed {
		return s.errorPage(ErrInvalidRequest, "this application has to push its authorization requests first"), nil
	}

	// Until the redirect URI is known to be the application's, no error may
	// be sent to it.
	if page := s.redirectProblem(app, p); page != "" {
		return s.errorPage(ErrInvalidRequest, page), nil
	}

	code, description, err := s.requestProblem(ctx, app, p)
	if err != nil {
		return "", err
	}
	if code != "" {
		return s.redirectError(p.RedirectURI, p.State, code, description), nil
	}

	prompts := strings.Fields(p.Prompt)
	maxAge := -1
	if p.MaxAge != "" {
		maxAge, _ = strconv.Atoi(p.MaxAge)
	}

	back := func(code, description string) (string, error) {
		return s.redirectError(p.RedirectURI, p.State, code, description), nil
	}

	now := s.now()

	if session != nil {
		tooOld := maxAge >= 0 && now.Sub(session.Record.AuthenticatedAt) > time.Duration(maxAge)*time.Second
		if tooOld || slices.Contains(prompts, "login") {
			session = nil
		}
	}

	// A session made by a looser flow must still meet this application's flow,
	// or the user is sent to sign in again.
	if session != nil {
		flow, err := s.store.EffectiveLoginFlow(ctx, app)
		if err != nil {
			return "", err
		}
		if !sessionSatisfies(flow, session, now) {
			session = nil
		}
	}

	if session == nil && slices.Contains(prompts, "none") {
		return back(ErrLoginRequired, "the user is not signed in")
	}

	handle, hash, err := model.NewSecret()
	if err != nil {
		return "", err
	}

	req := &model.AuthorizationRequest{
		HandleHash:          hash,
		ApplicationID:       app.ID,
		RedirectURI:         p.RedirectURI,
		Scope:               strings.Join(strings.Fields(p.Scope), " "),
		State:               p.State,
		Nonce:               p.Nonce,
		CodeChallenge:       p.CodeChallenge,
		CodeChallengeMethod: p.CodeChallengeMethod,
		Audience:            p.Audience,
		LoginHint:           p.LoginHint,
		ForceConsent:        slices.Contains(prompts, "consent"),
		DPoPKey:             p.DPoPKey,
		ExpiresAt:           now.Add(model.AuthorizationRequestLifetime),
	}
	req.Application = app

	if err := s.store.CreateAuthorizationRequest(ctx, req); err != nil {
		return "", err
	}

	if session != nil {
		if slices.Contains(prompts, "none") {
			page, err := s.stepBefore(ctx, req, session.User)
			if err != nil {
				return "", err
			}
			if page != "" {
				return back(silentRefusal(page))
			}
		}

		return s.finish(ctx, req, handle, session.User, &session.Record)
	}

	return withQuery(s.accountURL+PageLogin, url.Values{"request": {handle}}), nil
}

// redirectProblem is what is wrong before any error may be sent to the
// redirect URI: it is missing, not the application's, or the request is
// oversized. "" when there is nothing.
func (s *Service) redirectProblem(app *model.Application, p AuthorizeParams) string {
	switch {
	case p.RedirectURI == "":
		return "redirect_uri is required"
	case !app.AllowsRedirect(p.RedirectURI):
		return "redirect_uri is not registered for this application; it has to match exactly"
	}

	// Stored values have size limits and must be valid UTF-8; an oversized
	// request goes to the error page rather than being echoed back.
	return p.oversized()
}

// requestProblem checks an authorization request against its application, as
// the authorization endpoint and the pushed one both do once the redirect URI
// is known to be good. It returns an OAuth error code and description, or "".
func (s *Service) requestProblem(ctx context.Context, app *model.Application, p AuthorizeParams) (code, description string, err error) {
	switch {
	case !app.IsEnabled:
		return ErrUnauthorizedClient, "the application is disabled", nil
	case p.ResponseType != "code":
		return ErrUnsupportedResponseType, "response_type must be code", nil
	case p.ResponseMode != "" && p.ResponseMode != "query":
		return ErrInvalidRequest, "response_mode must be query", nil
	case !slices.Contains(app.GrantTypes, model.GrantAuthorizationCode):
		return ErrUnauthorizedClient, "the application does not have the authorization_code grant", nil
	case p.CodeChallenge == "" && app.MustUsePKCE():
		return ErrInvalidRequest, "code_challenge is required: this application must use PKCE", nil
	case p.CodeChallenge != "" && p.CodeChallengeMethod != model.PKCES256:
		return ErrInvalidRequest, "code_challenge_method must be S256", nil
	case p.CodeChallenge != "" && (len(p.CodeChallenge) != 43):
		return ErrInvalidRequest, "code_challenge must be the base64url SHA-256 of the code verifier", nil
	case strings.TrimSpace(p.Scope) == "":
		return ErrInvalidScope, "scope is required", nil
	case p.DPoPKey != "" && !isThumbprint(p.DPoPKey):
		return ErrInvalidRequest, "dpop_jkt must be the base64url SHA-256 thumbprint of the DPoP key", nil
	}

	if p.Audience != "" {
		if _, err := s.store.APIByIdentifier(ctx, p.Audience); errors.Is(err, store.ErrNotFound) {
			return ErrInvalidRequest, errUnknownAudience.Description, nil
		} else if err != nil {
			return "", "", err
		}
	}

	prompts := strings.Fields(p.Prompt)
	if slices.Contains(prompts, "none") && len(prompts) > 1 {
		return ErrInvalidRequest, "prompt=none cannot be combined with other values", nil
	}

	if p.MaxAge != "" {
		if n, err := strconv.Atoi(p.MaxAge); err != nil || n < 0 {
			return ErrInvalidRequest, "max_age must be a number of seconds", nil
		}
	}

	return "", "", nil
}

// sessionSatisfies reports whether this flow would have made the session.
// Sessions do not record the steps they passed, so a flow with the emailed-code
// step is never satisfied by an existing session.
func sessionSatisfies(flow *model.LoginFlow, session *Session, now time.Time) bool {
	switch {
	case !flow.AllowSignIn:
		return false
	case flow.RequireVerifiedEmail && !session.User.IsEmailVerified:
		return false
	case now.Sub(session.Record.AuthenticatedAt) > time.Duration(flow.SessionLifetimeHours)*time.Hour:
		return false
	case flow.Offers(model.StepEmailCode):
		return false
	}

	return true
}

type PendingRequest struct {
	Application *model.Application
	LoginHint   string
	ExpiresAt   time.Time
}

func (s *Service) Pending(ctx context.Context, handle string) (*PendingRequest, error) {
	req, err := s.pending(ctx, handle)
	if err != nil {
		return nil, err
	}

	return &PendingRequest{Application: req.Application, LoginHint: req.LoginHint, ExpiresAt: req.ExpiresAt}, nil
}

func (s *Service) pending(ctx context.Context, handle string) (*model.AuthorizationRequest, error) {
	if handle == "" {
		return nil, ErrRequestExpired
	}

	req, err := s.store.AuthorizationRequestByHandle(ctx, model.HashSecret(handle))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, ErrRequestExpired
	case err != nil:
		return nil, err
	}

	if !req.Usable(s.now()) || req.Application == nil {
		return nil, ErrRequestExpired
	}

	return req, nil
}

func (s *Service) Continue(ctx context.Context, handle string, session *Session) (string, error) {
	req, err := s.pending(ctx, handle)
	if err != nil {
		return "", err
	}

	return s.finish(ctx, req, handle, session.User, &session.Record)
}

// finish issues the code for a sign-in whose user is known, or sends the
// browser to the Agreements or Consent page first while one of them is still
// to be passed. handle names the request to those pages.
func (s *Service) finish(ctx context.Context, req *model.AuthorizationRequest, handle string, user *model.User, session *model.UserSession) (string, error) {
	app := req.Application

	// Redirect URIs and PKCE are checked again: an administrator may have
	// changed them since the request was made.
	if !app.AllowsRedirect(req.RedirectURI) {
		return s.errorPage(ErrInvalidRequest, "redirect_uri is no longer registered for this application"), nil
	}
	if app.MustUsePKCE() && req.CodeChallenge == "" {
		return s.redirectError(req.RedirectURI, req.State, ErrInvalidRequest, "this application now requires PKCE; start the sign-in again"), nil
	}

	// Evaluate now, as the token endpoint will, so a user who cannot get a
	// token is refused before a code is issued.
	token, err := s.evaluate(ctx, app, user, strings.Fields(req.Scope), req.Audience)
	if errors.Is(err, errUnknownAudience) {
		return s.redirectError(req.RedirectURI, req.State, ErrAccessDenied, "no API has the identifier "+req.Audience), nil
	}
	if err != nil {
		return "", err
	}
	if !token.Issued {
		return s.redirectError(req.RedirectURI, req.State, ErrAccessDenied, token.Reason), nil
	}

	page, err := s.stepBefore(ctx, req, user)
	if err != nil {
		return "", err
	}
	if page != "" {
		return s.stepPage(page, handle), nil
	}

	now := s.now()
	if err := s.store.CompleteAuthorizationRequest(ctx, req, now); errors.Is(err, store.ErrAlreadyUsed) {
		return "", ErrRequestExpired
	} else if err != nil {
		return "", err
	}

	code, hash, err := model.NewSecret()
	if err != nil {
		return "", err
	}

	err = s.store.CreateAuthorizationCode(ctx, &model.AuthorizationCode{
		CodeHash:            hash,
		ApplicationID:       app.ID,
		UserID:              user.ID,
		SessionID:           &session.ID,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		Nonce:               req.Nonce,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		Audience:            req.Audience,
		AuthenticatedAt:     session.AuthenticatedAt,
		Methods:             session.Methods,
		DPoPKey:             req.DPoPKey,
		ExpiresAt:           now.Add(model.AuthorizationCodeLifetime),
	})
	if err != nil {
		return "", err
	}

	// iss tells the client which server answered, so a mix-up attack across
	// two providers is caught (RFC 9207).
	return withQuery(req.RedirectURI, url.Values{"code": {code}, "state": {req.State}, "iss": {s.issuer}}), nil
}

func (s *Service) redirectError(redirectURI, state, code, description string) string {
	return withQuery(redirectURI, url.Values{
		"error":             {code},
		"error_description": {description},
		"state":             {state},
		"iss":               {s.issuer},
	})
}

var errUnknownAudience = oauthError(ErrInvalidRequest, "audience: no API has this identifier")

func (s *Service) evaluate(ctx context.Context, app *model.Application, user *model.User, scopes []string, audience string) (model.TokenPreview, error) {
	request := model.TokenRequest{
		Application: *app,
		User:        user,
		Requested:   scopes,
		Issuer:      s.issuer,
		Now:         s.now(),
	}

	if user != nil {
		roles, err := s.store.EffectiveRoles(ctx, user)
		if err != nil {
			return model.TokenPreview{}, err
		}
		request.Roles = roles
	}

	if audience != "" {
		access, err := s.store.AudienceFor(ctx, app.ID, audience)
		if errors.Is(err, store.ErrNotFound) {
			return model.TokenPreview{}, errUnknownAudience
		}
		if err != nil {
			return model.TokenPreview{}, err
		}

		request.API = access.API
		request.Authorized = access.Authorized
		request.Allowed = access.Allowed
	}

	return model.EvaluateToken(request), nil
}
