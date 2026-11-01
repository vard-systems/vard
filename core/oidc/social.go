package oidc

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/url"
	"strings"
	"time"

	"vard/core/model"
	"vard/core/store"
	"vard/i18n"
)

const (
	PathSocialStart    = "/oauth2/social/:slug/start"
	PathSocialCallback = "/oauth2/social/:slug/callback"
)

const socialTimeout = 15 * time.Second

// SocialButton is a provider as a sign-in page shows it.
type SocialButton struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

func (s *Service) SocialButtons(ctx context.Context) ([]SocialButton, error) {
	stored, err := s.store.SocialButtons(ctx)
	if err != nil {
		return nil, err
	}

	buttons := make([]SocialButton, 0, len(stored))
	for _, button := range stored {
		buttons = append(buttons, SocialButton(button))
	}

	return buttons, nil
}

func (s *Service) StartSocial(ctx context.Context, slug, request, next string) (string, string, error) {
	provider, err := s.socialProvider(ctx, slug)
	if err != nil {
		return "", "", err
	}

	flow, err := s.flowFor(ctx, request)
	if err != nil {
		return "", "", err
	}
	if !flow.Offers(model.StepSocial) {
		return "", "", ErrSocialNotOffered
	}
	spec := provider.Spec()

	state, stateHash, err := model.NewSecret()
	if err != nil {
		return "", "", err
	}

	values := url.Values{
		"response_type": {"code"},
		"client_id":     {provider.ClientID},
		"redirect_uri":  {provider.CallbackURL(s.issuer)},
		"state":         {state},
	}

	if scopes := provider.AskedScopes(); len(scopes) > 0 {
		values.Set("scope", strings.Join(scopes, " "))
	}

	// PKCE binds the code to this sign-in, so a code taken out of a redirect
	// cannot be spent by anyone else (RFC 9700 section 2.1.1).
	var verifier string
	if spec.PKCE {
		if verifier, _, err = model.NewSecret(); err != nil {
			return "", "", err
		}

		values.Set("code_challenge", challengeFor(verifier))
		values.Set("code_challenge_method", model.PKCES256)
	}

	for key, value := range spec.AuthorizeParams {
		values.Set(key, value)
	}

	// The identity read out of an id_token has to be this sign-in's, not one
	// issued to the same client for somebody else (Core 3.1.2.1).
	if identityInIDToken(provider) {
		values.Set("nonce", s.socialNonce(state))
	}

	login := model.SocialLogin{
		StateHash:  stateHash,
		ProviderID: provider.ID,
		Verifier:   verifier,
		Request:    request,
		Next:       next,
		ExpiresAt:  s.now().Add(model.SocialLoginLifetime),
	}
	if err := s.store.CreateSocialLogin(ctx, &login); err != nil {
		return "", "", err
	}

	authorize, _, _ := provider.Endpoints()

	return withQuery(authorize, values), state, nil
}

type SocialResult struct {
	SignIn   *SignInResult
	Request  string
	Next     string
	Provider string
}

func (s *Service) CompleteSocial(ctx context.Context, slug, code, state, binding string, client Client) (*SocialResult, error) {
	provider, err := s.socialProvider(ctx, slug)
	if err != nil {
		return nil, err
	}

	if code == "" || state == "" {
		return nil, ErrSocialExpired
	}

	// The answer must return to the browser that started the sign-in, or a
	// stolen state and code could sign a victim's browser into the attacker's
	// account (RFC 6749 10.12).
	if subtle.ConstantTimeCompare([]byte(binding), []byte(state)) != 1 {
		return nil, ErrSocialExpired
	}

	login, err := s.store.TakeSocialLogin(ctx, model.HashSecret(state), s.now())
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, ErrSocialExpired
	case err != nil:
		return nil, err
	}

	if login.ProviderID != provider.ID {
		return nil, ErrSocialExpired
	}

	token, err := s.exchangeSocialCode(ctx, provider, code, login.Verifier)
	if err != nil {
		return nil, err
	}

	identity, err := s.socialIdentity(ctx, provider, token, s.socialNonce(state))
	if err != nil {
		return nil, err
	}

	result, err := s.signInWithIdentity(ctx, provider, identity, login.Request, client)
	if err != nil {
		return nil, err
	}

	return &SocialResult{
		SignIn:   result,
		Request:  login.Request,
		Next:     login.Next,
		Provider: provider.Name,
	}, nil
}

func (s *Service) socialProvider(ctx context.Context, slug string) (*model.SocialProvider, error) {
	provider, err := s.store.SocialProviderBySlug(ctx, slug)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, ErrSocialUnknown
	case err != nil:
		return nil, err
	case !provider.IsEnabled:
		return nil, ErrSocialUnknown
	}

	return provider, nil
}

func (s *Service) SocialLanding(ctx context.Context, result *SocialResult) string {
	if result.Request != "" {
		location, err := s.Continue(ctx, result.Request, result.SignIn.Session)
		if err == nil {
			return location
		}

		s.log.Info("a sign-in request expired while the user was at a provider",
			"provider", result.Provider, "error", err)
	}

	return s.accountLanding(result.Next)
}

// SecondFactorLanding is the second-factor page, for a provider sign-in
// that waits for the user's second factor. It carries on to where the
// sign-in was going once that is given.
func (s *Service) SecondFactorLanding(result *SocialResult) string {
	values := url.Values{}
	if result.Request != "" {
		values.Set("request", result.Request)
	}
	if safePath(result.Next) {
		values.Set("next", result.Next)
	}

	return withQuery(s.accountURL+PageSecondFactor, values)
}

func (s *Service) accountLanding(next string) string {
	if safePath(next) {
		return s.accountURL + next
	}

	return s.accountURL + "/"
}

// safePath reports whether `next` is a path on the account app only, by the
// same rule as the app's safeNext: one leading slash, no second slash or
// backslash, and no control characters (browsers strip tabs and newlines before
// parsing).
func safePath(next string) bool {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, `/\`) {
		return false
	}

	for _, r := range next {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}

	return true
}

func (s *Service) SocialErrorPage(err error) string {
	reason := ErrSocialUpstream

	var known *Problem
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		reason = ErrSocialBlocked
	case errors.Is(err, ErrEmailNotVerified):
		reason = ErrEmailNotVerified
	case errors.As(err, &known) && strings.HasPrefix(known.Code, "social_"):
		reason = known
	}

	description, ok := i18n.Text(i18n.ID, "error."+reason.Code)
	if !ok {
		description = reason.Code
	}

	return withQuery(s.accountURL+PageError, url.Values{
		"error":             {SocialErrorCode},
		"reason":            {reason.Code},
		"error_description": {description},
	})
}

const SocialErrorCode = "social_sign_in_failed"

func IsSocialFailure(err error) bool {
	for _, known := range []error{
		ErrSocialUnknown,
		ErrSocialExpired,
		ErrSocialUpstream,
		ErrSocialNoEmail,
		ErrSocialLinkRefused,
		ErrSocialEmailUnverified,
		ErrSocialRegistrationClosed,
		ErrSocialSSORequired,
		ErrInvalidCredentials,
	} {
		if errors.Is(err, known) {
			return true
		}
	}

	return false
}
