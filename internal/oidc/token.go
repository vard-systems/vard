package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"loginer/internal/jose"
	"loginer/internal/model"
	"loginer/internal/store"
)

// idTokenAlgorithm signs every ID token. RS256 is the one algorithm every
// OpenID Connect client must accept; access tokens follow their API's choice.
const idTokenAlgorithm = jose.RS256

type ClientAuth struct {
	ID     string
	Secret string
	Method model.AuthMethod
}

func (s *Service) authenticate(ctx context.Context, auth ClientAuth) (*model.Application, error) {
	if auth.ID == "" {
		return nil, oauthError(ErrInvalidClient, "client authentication is required")
	}

	app, err := s.store.ApplicationByClientID(ctx, auth.ID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, oauthError(ErrInvalidClient, "unknown client")
	}
	if err != nil {
		return nil, err
	}

	registered := app.TokenAuthMethod
	switch {
	case registered == model.AuthNone && auth.Method != model.AuthNone:
		return nil, oauthError(ErrInvalidClient, "this is a public client: send client_id alone, with no secret")
	case registered != model.AuthNone && auth.Method != registered:
		return nil, oauthError(ErrInvalidClient, "this client authenticates with "+string(registered))
	case registered != model.AuthNone && !app.CheckSecret(auth.Secret):
		return nil, oauthError(ErrInvalidClient, "client authentication failed")
	}

	if !app.IsEnabled {
		return nil, oauthError(ErrUnauthorizedClient, "the application is disabled")
	}

	return app, nil
}

type TokenParams struct {
	Client       ClientAuth
	GrantType    string
	Code         string
	RedirectURI  string
	CodeVerifier string
	RefreshToken string
	DeviceCode   string
	Scope        string
	Audience     string
	// DPoP is the request's DPoP proof, "" when it sent none.
	DPoP string

	// dpopKey is the thumbprint of the key the proof showed, once checked.
	dpopKey string
}

// TokenResponse is a successful token response (RFC 6749 section 5.1).
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope"`
}

func (s *Service) Token(ctx context.Context, p TokenParams) (*TokenResponse, error) {
	app, err := s.authenticate(ctx, p.Client)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(model.GrantTypes, p.GrantType) {
		return nil, oauthError(ErrUnsupportedGrantType, "grant_type must be one of: "+strings.Join(model.GrantTypes, ", "))
	}
	if !slices.Contains(app.GrantTypes, p.GrantType) {
		return nil, oauthError(ErrUnauthorizedClient, "the application does not have the "+p.GrantType+" grant")
	}

	if p.DPoP != "" {
		if p.dpopKey, err = s.checkDPoP(ctx, p.DPoP, "POST", s.issuer+PathToken, ""); err != nil {
			return nil, err
		}
	}

	switch p.GrantType {
	case model.GrantAuthorizationCode:
		return s.exchangeCode(ctx, app, p)
	case model.GrantRefreshToken:
		return s.refresh(ctx, app, p)
	case model.GrantDeviceCode:
		return s.deviceToken(ctx, app, p)
	default:
		return s.clientCredentials(ctx, app, p)
	}
}

func (s *Service) exchangeCode(ctx context.Context, app *model.Application, p TokenParams) (*TokenResponse, error) {
	if p.Code == "" {
		return nil, oauthError(ErrInvalidRequest, "code is required")
	}

	now := s.now()
	code, err := s.store.ClaimAuthorizationCode(ctx, model.HashSecret(p.Code), app.ID, now)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, oauthError(ErrInvalidGrant, "the authorization code is not valid")
	case errors.Is(err, store.ErrAlreadyUsed):
		// Found but not claimed: another client's code is left alone so its
		// owner can still spend it; this client's own code presented twice
		// means it leaked, so what the first exchange issued is revoked.
		if code.ApplicationID != app.ID {
			return nil, oauthError(ErrInvalidGrant, "the authorization code was issued to another client")
		}
		if err := s.store.RevokeRefreshTokensForCode(ctx, code.ID, now); err != nil {
			return nil, err
		}
		return nil, oauthError(ErrInvalidGrant, "the authorization code has already been used")
	case err != nil:
		return nil, err
	}

	switch {
	case !now.Before(code.ExpiresAt):
		return nil, oauthError(ErrInvalidGrant, "the authorization code has expired")
	case p.RedirectURI != code.RedirectURI:
		return nil, oauthError(ErrInvalidGrant, "redirect_uri does not match the authorization request")
	case !slices.Contains(app.RedirectURIs, code.RedirectURI):
		return nil, oauthError(ErrInvalidGrant, "redirect_uri is no longer registered for this application")
	case app.RequirePKCE && code.CodeChallenge == "":
		return nil, oauthError(ErrInvalidGrant, "this application now requires PKCE")
	case code.CodeChallenge != "" && !model.VerifyPKCE(code.CodeChallenge, p.CodeVerifier):
		return nil, oauthError(ErrInvalidGrant, "code_verifier does not match the code_challenge")
	case code.CodeChallenge == "" && p.CodeVerifier != "":
		return nil, oauthError(ErrInvalidGrant, "code_verifier was sent, but the authorization request had no code_challenge")
	}

	user, err := s.store.User(ctx, code.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, oauthError(ErrInvalidGrant, "the user no longer exists")
	}
	if err != nil {
		return nil, err
	}

	// The session the code came from must still be alive, or a code minted just
	// before a password reset would start a fresh refresh-token family for
	// whoever the reset shut out.
	if code.SessionID != nil {
		session, err := s.store.UserSession(ctx, *code.SessionID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			return nil, oauthError(ErrInvalidGrant, "the session the code was issued in has ended")
		case err != nil:
			return nil, err
		case !session.Active(now):
			return nil, oauthError(ErrInvalidGrant, "the session the code was issued in has ended")
		}
	}

	response, err := s.issue(ctx, grant{
		app:      app,
		user:     user,
		scopes:   strings.Fields(code.Scope),
		audience: code.Audience,
		nonce:    code.Nonce,
		authTime: code.AuthenticatedAt,
		session:  code.SessionID,
		codeID:   &code.ID,
		dpopKey:  p.dpopKey,
		refusal:  ErrInvalidGrant,
	})
	if err != nil {
		return nil, err
	}

	// The tokens are already issued and the code spent, so failing here would
	// only lose them; the application is then just not told of the logout.
	if code.SessionID != nil && response.IDToken != "" {
		if err := s.store.RememberSessionApplication(ctx, *code.SessionID, app.ID); err != nil {
			s.log.Error("recording the application a session signed in failed", "error", err)
		}
	}

	return response, nil
}

func (s *Service) refresh(ctx context.Context, app *model.Application, p TokenParams) (*TokenResponse, error) {
	if p.RefreshToken == "" {
		return nil, oauthError(ErrInvalidRequest, "refresh_token is required")
	}

	now := s.now()
	old, err := s.store.RefreshTokenByHash(ctx, model.HashSecret(p.RefreshToken))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, oauthError(ErrInvalidGrant, "the refresh token is not valid")
	case err != nil:
		return nil, err
	}

	if old.ApplicationID != app.ID {
		return nil, oauthError(ErrInvalidGrant, "the refresh token was issued to another client")
	}

	if old.RevokedAt != nil {
		// A replaced token presented again: one of the two presenters is not
		// the client. The whole family goes, so the thief's copy dies too.
		if err := s.store.RevokeRefreshFamily(ctx, old.FamilyID, now); err != nil {
			return nil, err
		}
		return nil, oauthError(ErrInvalidGrant, "the refresh token has been revoked")
	}

	if !old.Usable(now) {
		return nil, oauthError(ErrInvalidGrant, "the refresh token has expired")
	}

	// A family bound to a key goes on only with a proof from that key.
	if old.DPoPKey != "" && p.dpopKey != old.DPoPKey {
		return nil, oauthError(ErrInvalidGrant, "the refresh token is bound to a DPoP key; send a proof from it")
	}

	original := strings.Fields(old.Scope)
	scopes := original
	if p.Scope != "" {
		scopes = strings.Fields(p.Scope)
		for _, scope := range scopes {
			if !slices.Contains(original, scope) {
				return nil, oauthError(ErrInvalidScope, "a refresh can only narrow the original scope; "+scope+" was not granted")
			}
		}
	}

	user, err := s.store.User(ctx, old.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, oauthError(ErrInvalidGrant, "the user no longer exists")
	}
	if err != nil {
		return nil, err
	}

	return s.issue(ctx, grant{
		app:      app,
		user:     user,
		scopes:   scopes,
		audience: old.Audience,
		authTime: old.AuthenticatedAt,
		session:  old.SessionID,
		replaces: old,
		dpopKey:  p.dpopKey,
		refusal:  ErrInvalidGrant,
	})
}

func (s *Service) clientCredentials(ctx context.Context, app *model.Application, p TokenParams) (*TokenResponse, error) {
	if app.TokenAuthMethod == model.AuthNone {
		return nil, oauthError(ErrUnauthorizedClient, "a public client cannot use client_credentials")
	}

	return s.issue(ctx, grant{
		app:      app,
		scopes:   strings.Fields(p.Scope),
		audience: p.Audience,
		dpopKey:  p.dpopKey,
		refusal:  ErrUnauthorizedClient,
	})
}

type grant struct {
	app      *model.Application
	user     *model.User
	scopes   []string
	audience string
	nonce    string
	authTime time.Time
	session  *uuid.UUID
	codeID   *uuid.UUID
	replaces *model.RefreshToken
	// dpopKey binds the tokens to a key, when the request proved one.
	dpopKey string
	refusal string
}

func (s *Service) issue(ctx context.Context, g grant) (*TokenResponse, error) {
	evaluated, err := s.evaluate(ctx, g.app, g.user, g.scopes, g.audience)
	if err != nil {
		return nil, err
	}
	if !evaluated.Issued {
		return nil, oauthError(g.refusal, evaluated.Reason)
	}

	jti, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	access := evaluated.AccessToken
	access["jti"] = jti.String()
	if g.dpopKey != "" {
		access["cnf"] = map[string]any{"jkt": g.dpopKey}
	}
	if g.user != nil {
		access["auth_time"] = g.authTime.Unix()
	}

	algorithm, _ := evaluated.AccessTokenHeader["alg"].(string)
	accessKey, ok := s.keys.signing(algorithm)
	if !ok {
		return nil, errors.New("oidc: no signing key for " + algorithm)
	}

	accessToken, err := jose.Sign(accessKey, "at+jwt", access)
	if err != nil {
		return nil, err
	}

	granted, _ := access["scope"].(string)
	iat, _ := access["iat"].(int64)
	exp, _ := access["exp"].(int64)

	response := &TokenResponse{
		AccessToken: accessToken,
		TokenType:   tokenType(g.dpopKey),
		ExpiresIn:   exp - iat,
		Scope:       granted,
	}

	if evaluated.IDToken != nil {
		id := evaluated.IDToken
		id["auth_time"] = g.authTime.Unix()
		id["at_hash"] = jose.HalfHash(accessToken)
		id["azp"] = g.app.ClientID
		if g.nonce != "" {
			id["nonce"] = g.nonce
		}
		if g.session != nil {
			id["sid"] = g.session.String()
		}

		idKey, ok := s.keys.signing(idTokenAlgorithm)
		if !ok {
			return nil, errors.New("oidc: no signing key for " + idTokenAlgorithm)
		}

		if response.IDToken, err = jose.Sign(idKey, "JWT", id); err != nil {
			return nil, err
		}
	}

	// A refresh always rotates, even if offline_access was narrowed away, so a
	// stolen token cannot be replayed without reaching reuse detection. Other
	// grants get a refresh token only with offline_access.
	if g.user != nil && (g.replaces != nil || slices.Contains(strings.Fields(granted), model.ScopeOfflineAccess)) {
		token, err := s.newRefreshToken(ctx, g, granted)
		if err != nil {
			return nil, err
		}
		response.RefreshToken = token
	}

	return response, nil
}

func (s *Service) newRefreshToken(ctx context.Context, g grant, scope string) (string, error) {
	now := s.now()

	token, hash, err := model.NewSecret()
	if err != nil {
		return "", err
	}

	next := &model.RefreshToken{
		TokenHash:       hash,
		ApplicationID:   g.app.ID,
		UserID:          g.user.ID,
		CodeID:          g.codeID,
		SessionID:       g.session,
		DPoPKey:         refreshBinding(g),
		Scope:           scope,
		Audience:        g.audience,
		AuthenticatedAt: g.authTime,
		ExpiresAt:       now.Add(time.Duration(g.app.RefreshTokenLifetime) * time.Second),
	}

	if g.replaces == nil {
		if next.FamilyID, err = uuid.NewV7(); err != nil {
			return "", err
		}
		return token, s.store.CreateRefreshToken(ctx, next)
	}

	next.FamilyID = g.replaces.FamilyID
	next.CodeID = g.replaces.CodeID
	next.DPoPKey = g.replaces.DPoPKey
	next.ExpiresAt = g.replaces.ExpiresAt

	err = s.store.RotateRefreshToken(ctx, g.replaces, next, now)
	if errors.Is(err, store.ErrAlreadyUsed) {
		// Another request rotated it first: the same replay as above.
		if err := s.store.RevokeRefreshFamily(ctx, g.replaces.FamilyID, now); err != nil {
			return "", err
		}
		return "", oauthError(ErrInvalidGrant, "the refresh token has been revoked")
	}

	return token, err
}

// tokenType is Bearer, or DPoP for a token bound to a key.
func tokenType(dpopKey string) string {
	if dpopKey != "" {
		return TokenTypeDPoP
	}
	return "Bearer"
}

// refreshBinding is the key a new family is bound to: a public client's, since
// a confidential one is bound by its own credentials (RFC 9449 §5).
func refreshBinding(g grant) string {
	if g.app.Type.Public() {
		return g.dpopKey
	}
	return ""
}

type accessClaims struct {
	Issuer   string      `json:"iss"`
	Subject  string      `json:"sub"`
	Audience []string    `json:"aud"`
	ClientID string      `json:"client_id"`
	Scope    string      `json:"scope"`
	IssuedAt json.Number `json:"iat"`
	Expiry   json.Number `json:"exp"`
}

// verifyAccessToken checks an access token this server issued: signed by one of
// its keys, an access token and not an ID token, from this issuer, unexpired.
func (s *Service) verifyAccessToken(ctx context.Context, token string) (*accessClaims, map[string]any, error) {
	var raw map[string]any
	header, err := jose.Verify(token, s.keys.lookup(ctx), &raw)
	if err != nil || header.Type != "at+jwt" {
		return nil, nil, jose.ErrInvalid
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, nil, err
	}

	var claims accessClaims
	if err := json.Unmarshal(encoded, &claims); err != nil {
		return nil, nil, jose.ErrInvalid
	}

	exp, err := claims.Expiry.Int64()
	if err != nil || claims.Issuer != s.issuer || s.now().Unix() >= exp {
		return nil, nil, jose.ErrInvalid
	}

	return &claims, raw, nil
}

func (s *Service) UserInfo(ctx context.Context, accessToken string, how Presentation) (map[string]any, error) {
	if accessToken == "" {
		return nil, oauthError(ErrInvalidToken, "an access token is required")
	}

	claims, raw, err := s.verifyAccessToken(ctx, accessToken)
	if err != nil {
		return nil, oauthError(ErrInvalidToken, "the access token is invalid or has expired")
	}
	if err := s.presented(ctx, accessToken, raw, how); err != nil {
		return nil, err
	}

	scopes := strings.Fields(claims.Scope)
	if !slices.Contains(scopes, model.ScopeOpenID) {
		return nil, oauthError(ErrInsufficientScope, "the access token was not granted the openid scope")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, oauthError(ErrInvalidToken, "the access token is not for a user")
	}

	app, err := s.store.ApplicationByClientID(ctx, claims.ClientID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, oauthError(ErrInvalidToken, "the application no longer exists")
	}
	if err != nil {
		return nil, err
	}

	user, err := s.store.User(ctx, userID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, oauthError(ErrInvalidToken, "the user no longer exists")
	}
	if err != nil {
		return nil, err
	}

	evaluated, err := s.evaluate(ctx, app, user, scopes, "")
	if err != nil {
		return nil, err
	}
	if !evaluated.Issued || evaluated.IDToken == nil {
		return nil, oauthError(ErrInvalidToken, evaluated.Reason)
	}

	info := evaluated.IDToken
	for _, claim := range []string{"iss", "aud", "iat", "exp"} {
		delete(info, claim)
	}

	return info, nil
}

// Revoke revokes a refresh token and its family (RFC 7009). Access tokens are
// self-contained JWTs and simply expire; revoking one, or an unknown token,
// still succeeds as the RFC requires.
func (s *Service) Revoke(ctx context.Context, client ClientAuth, token string) error {
	app, err := s.authenticate(ctx, client)
	if err != nil {
		return err
	}

	if token == "" {
		return oauthError(ErrInvalidRequest, "token is required")
	}

	refresh, err := s.store.RefreshTokenByHash(ctx, model.HashSecret(token))
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	// Another client's token is left alone, and the answer does not say it
	// exists.
	if refresh.ApplicationID != app.ID {
		return nil
	}

	return s.store.RevokeRefreshFamily(ctx, refresh.FamilyID, s.now())
}

// Introspect reports whether a token is active (RFC 7662). Only confidential
// clients may ask, and only about their own tokens; any other client's token
// reads as inactive.
func (s *Service) Introspect(ctx context.Context, client ClientAuth, token string) (map[string]any, error) {
	app, err := s.authenticate(ctx, client)
	if err != nil {
		return nil, err
	}
	if app.TokenAuthMethod == model.AuthNone {
		return nil, oauthError(ErrInvalidClient, "introspection needs a confidential client")
	}

	inactive := map[string]any{"active": false}
	if token == "" {
		return inactive, nil
	}

	if claims, raw, err := s.verifyAccessToken(ctx, token); err == nil {
		// Another client's token reads as inactive: the answer carries the
		// subject, scope and roles.
		if claims.ClientID != app.ClientID {
			return inactive, nil
		}

		// A user turned off or deleted since is no longer vouched for, as
		// userinfo and the refresh grant already say; a service token's
		// subject is the client itself.
		if claims.Subject != claims.ClientID {
			id, err := uuid.Parse(claims.Subject)
			if err != nil {
				return inactive, nil
			}
			user, err := s.store.User(ctx, id)
			if errors.Is(err, store.ErrNotFound) {
				return inactive, nil
			}
			if err != nil {
				return nil, err
			}
			if !user.IsActive {
				return inactive, nil
			}
		}

		out := map[string]any{"active": true, "token_type": tokenType(boundKey(raw))}
		maps.Copy(out, raw)
		return out, nil
	}

	refresh, err := s.store.RefreshTokenByHash(ctx, model.HashSecret(token))
	if errors.Is(err, store.ErrNotFound) {
		return inactive, nil
	}
	if err != nil {
		return nil, err
	}

	if refresh.ApplicationID != app.ID || !refresh.Usable(s.now()) {
		return inactive, nil
	}

	// A token for a user who has since been turned off is spent, not live:
	// the refresh grant itself would refuse it, so introspection says so too.
	if user, err := s.store.User(ctx, refresh.UserID); err != nil || !user.IsActive {
		return inactive, nil
	}

	return map[string]any{
		"active":     true,
		"token_type": "refresh_token",
		"client_id":  app.ClientID,
		"sub":        refresh.UserID.String(),
		"scope":      refresh.Scope,
		"exp":        refresh.ExpiresAt.Unix(),
		"iat":        refresh.CreatedAt.Unix(),
		"iss":        s.issuer,
	}, nil
}
