package oidc

import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"loginer/i18n"
	"loginer/internal/jose"
	"loginer/internal/model"
	"loginer/internal/store"
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

	identity, err := s.socialIdentity(ctx, provider, token)
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

func challengeFor(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

type socialToken struct {
	AccessToken string
	IDToken     string
	Raw         map[string]any
}

func (s *Service) exchangeSocialCode(ctx context.Context, provider *model.SocialProvider, code, verifier string) (*socialToken, error) {
	_, tokenURL, _ := provider.Endpoints()

	secret, err := s.socialClientSecret(provider)
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"grant_type":   {model.GrantAuthorizationCode},
		"code":         {code},
		"redirect_uri": {provider.CallbackURL(s.issuer)},
		"client_id":    {provider.ClientID},
	}
	if verifier != "" {
		form.Set("code_verifier", verifier)
	}

	// Only ever one way at a time: a server that is sent both may refuse them
	// both (RFC 6749 section 2.3).
	basic := provider.TokenAuthMethod() == model.SocialTokenAuthBasic
	if !basic {
		form.Set("client_secret", secret)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	if basic {
		request.SetBasicAuth(url.QueryEscape(provider.ClientID), url.QueryEscape(secret))
	}

	body, err := s.callProvider(request, provider, "token")
	if err != nil {
		return nil, err
	}

	var answer map[string]any
	if err := json.Unmarshal(body, &answer); err != nil {
		s.log.Error("a social provider's token answer was not JSON", "provider", provider.Slug, "error", err)
		return nil, ErrSocialUpstream
	}

	token := &socialToken{
		AccessToken: stringClaim(answer, "access_token"),
		IDToken:     stringClaim(answer, "id_token"),
		Raw:         answer,
	}

	if token.AccessToken == "" && token.IDToken == "" {
		s.log.Error("a social provider gave no token",
			"provider", provider.Slug, "error", stringClaim(answer, "error"))
		return nil, ErrSocialUpstream
	}

	return token, nil
}

func (s *Service) socialClientSecret(provider *model.SocialProvider) (string, error) {
	if !provider.Spec().SignedSecret {
		secret, err := s.sealer.OpenBytes(provider.ClientSecret)
		if err != nil {
			return "", fmt.Errorf("unseal the client secret of %s: %w", provider.Slug, err)
		}

		return string(secret), nil
	}

	return s.appleClientSecret(provider)
}

func (s *Service) appleClientSecret(provider *model.SocialProvider) (string, error) {
	pemBytes, err := s.sealer.OpenBytes(provider.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("unseal the signing key of %s: %w", provider.Slug, err)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return "", fmt.Errorf("the signing key of %s is not PEM", provider.Slug)
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse the signing key of %s: %w", provider.Slug, err)
	}

	signer, ok := parsed.(crypto.Signer)
	if !ok {
		return "", fmt.Errorf("the signing key of %s cannot sign", provider.Slug)
	}

	now := s.now()

	return jose.Sign(
		jose.Key{ID: provider.KeyID, Algorithm: jose.ES256, Private: signer},
		"JWT",
		map[string]any{
			"iss": provider.TeamID,
			"sub": provider.ClientID,
			"aud": "https://appleid.apple.com",
			"iat": now.Unix(),
			"exp": now.Add(5 * time.Minute).Unix(),
		},
	)
}

type socialIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	FirstName     string
	LastName      string
}

func (s *Service) socialIdentity(ctx context.Context, provider *model.SocialProvider, token *socialToken) (*socialIdentity, error) {
	spec := provider.Spec()
	_, _, userInfoURL := provider.Endpoints()

	var claims map[string]any
	if spec.IdentityInIDToken || userInfoURL == "" {
		payload, err := s.idTokenClaims(provider, token.IDToken)
		if err != nil {
			return nil, err
		}
		claims = payload
	} else {
		payload, err := s.fetchSocialProfile(ctx, provider, token)
		if err != nil {
			return nil, err
		}
		claims = payload
	}

	identity := &socialIdentity{
		Subject:   claimString(claims, spec.Claims.Subject),
		Email:     claimString(claims, spec.Claims.Email),
		FirstName: claimString(claims, spec.Claims.FirstName),
		LastName:  claimString(claims, spec.Claims.LastName),
	}

	if identity.Email == "" && spec.EmailInTokenResponse {
		identity.Email = stringClaim(token.Raw, "email")
	}
	identity.EmailVerified = spec.AssumeEmailVerified || claimBool(claims, spec.Claims.EmailVerified)

	if identity.FirstName == "" && identity.LastName == "" {
		identity.FirstName, identity.LastName = splitName(claimString(claims, spec.Claims.FullName))
	}

	if identity.Subject == "" {
		s.log.Error("a social provider gave no subject", "provider", provider.Slug)
		return nil, ErrSocialUpstream
	}

	return identity, nil
}

// idTokenClaims reads an id_token straight from the provider's token endpoint
// over TLS, which OpenID Connect Core 3.1.3.7 accepts in place of a signature
// check. Its claims are still held to this client, the expected issuer and the
// clock.
func (s *Service) idTokenClaims(provider *model.SocialProvider, idToken string) (map[string]any, error) {
	if idToken == "" {
		s.log.Error("a social provider gave no id_token", "provider", provider.Slug)
		return nil, ErrSocialUpstream
	}

	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, ErrSocialUpstream
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrSocialUpstream
	}

	var claims map[string]any
	if err := decodeClaims(payload, &claims); err != nil {
		return nil, ErrSocialUpstream
	}

	if !audienceHas(claims["aud"], provider.ClientID) {
		s.log.Error("a social provider's id_token was for another client", "provider", provider.Slug)
		return nil, ErrSocialUpstream
	}

	// Kinds with a known provider must name its issuer; per-installation kinds
	// have none to check.
	if want := provider.Spec().IDTokenIssuer; want != "" && stringClaim(claims, "iss") != want {
		s.log.Error("a social provider's id_token named another issuer",
			"provider", provider.Slug, "issuer", stringClaim(claims, "iss"), "want", want)
		return nil, ErrSocialUpstream
	}

	if exp, ok := numberClaim(claims["exp"]); ok && s.now().After(time.Unix(int64(exp), 0)) {
		return nil, ErrSocialExpired
	}

	return claims, nil
}

func (s *Service) fetchSocialProfile(ctx context.Context, provider *model.SocialProvider, token *socialToken) (map[string]any, error) {
	spec := provider.Spec()
	_, _, userInfoURL := provider.Endpoints()

	values := url.Values{}
	for key, value := range spec.UserInfoParams {
		values.Set(key, value)
	}
	if spec.UserInfoTokenParam != "" {
		values.Set(spec.UserInfoTokenParam, token.AccessToken)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, withQuery(userInfoURL, values), nil)
	if err != nil {
		return nil, err
	}

	scheme := spec.UserInfoScheme
	if scheme == "" {
		scheme = "Bearer"
	}
	request.Header.Set("Authorization", scheme+" "+token.AccessToken)
	request.Header.Set("Accept", "application/json")

	body, err := s.callProvider(request, provider, "profile")
	if err != nil {
		return nil, err
	}

	var claims map[string]any
	if err := decodeClaims(body, &claims); err != nil {
		s.log.Error("a social provider's profile was not JSON", "provider", provider.Slug, "error", err)
		return nil, ErrSocialUpstream
	}

	return claims, nil
}

// decodeClaims keeps numbers as json.Number: as float64, ids above 2^53 round
// together and different people would share one subject.
func decodeClaims(payload []byte, claims *map[string]any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()

	return decoder.Decode(claims)
}

func (s *Service) callProvider(request *http.Request, provider *model.SocialProvider, what string) ([]byte, error) {
	client := s.social
	if client == nil {
		client = newFederationClient(socialTimeout, onLoopback(s.issuer))
	}

	response, err := client.Do(request)
	if err != nil {
		s.log.Error("a social provider could not be reached",
			"provider", provider.Slug, "request", what, "error", err)
		return nil, ErrSocialUpstream
	}
	defer response.Body.Close()

	// Enough of the answer to work with, and no more: a provider answering
	// with a stream is not going to be read into memory.
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, ErrSocialUpstream
	}

	if response.StatusCode != http.StatusOK {
		s.log.Error("a social provider refused",
			"provider", provider.Slug, "request", what,
			"status", response.StatusCode, "body", model.Truncate(string(body), 512))
		return nil, ErrSocialUpstream
	}

	return body, nil
}

func (s *Service) signInWithIdentity(
	ctx context.Context,
	provider *model.SocialProvider,
	who *socialIdentity,
	request string,
	client Client,
) (*SignInResult, error) {
	now := s.now()
	email := strings.ToLower(strings.TrimSpace(who.Email))

	flow, err := s.flowFor(ctx, request)
	if err != nil {
		return nil, err
	}

	// Closed means closed before any account is made, identity linked or role
	// synced — startSession would refuse at the end, but only after those.
	if !flow.AllowSignIn {
		return nil, ErrSignInClosed
	}

	identity, err := s.store.SocialIdentity(ctx, provider.ID, who.Subject)
	switch {
	case err == nil:
		user, err := s.store.User(ctx, identity.UserID)
		if err != nil {
			return nil, err
		}
		// The account's own address decides SSO enforcement, not the
		// provider's.
		if err := s.socialSSORequired(ctx, user.Email); err != nil {
			return nil, err
		}
		if !user.CanSignIn(now) {
			s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{
				"reason": "inactive", "provider": provider.Name,
			})
			return nil, ErrInvalidCredentials
		}

		if err := s.store.MarkIdentityUsed(ctx, identity, email, now); err != nil {
			return nil, err
		}

		return s.startSocialSession(ctx, user, provider, flow, request, client)
	case !errors.Is(err, store.ErrNotFound):
		return nil, err
	}

	if email == "" {
		return nil, ErrSocialNoEmail
	}

	if err := s.socialSSORequired(ctx, email); err != nil {
		return nil, err
	}

	// Linking to an existing account needs the address proved by the provider,
	// a provider trusted to prove it, and the account's own address verified;
	// otherwise whoever pre-registered the address would keep a password to it.
	user, err := s.store.UserByEmail(ctx, email)
	switch {
	case err == nil:
		if !who.EmailVerified || !provider.LinkVerifiedEmails || !user.IsEmailVerified {
			return nil, ErrSocialLinkRefused
		}
		if !user.CanSignIn(now) {
			return nil, ErrInvalidCredentials
		}

		if err := s.connectIdentity(ctx, user, provider, who, email, now); err != nil {
			return nil, err
		}
		s.record(ctx, user, user.Email, "user.identity_connected", client, map[string]any{
			"provider": provider.Name,
		})

		return s.startSocialSession(ctx, user, provider, flow, request, client)
	case !errors.Is(err, store.ErrNotFound):
		return nil, err
	}

	if !provider.AllowRegistration || !flow.AllowRegistration {
		return nil, ErrSocialRegistrationClosed
	}
	if err := s.socialRegistrationAllowed(ctx, request); err != nil {
		return nil, err
	}

	user = &model.User{
		Email:           email,
		IsEmailVerified: who.EmailVerified,
		FirstName:       model.Truncate(who.FirstName, 100),
		LastName:        model.Truncate(who.LastName, 100),
		IsActive:        true,
		Data:            map[string]any{},
	}

	roles, err := s.store.DefaultUserRoles(ctx)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	if err := s.store.CreateUser(ctx, user); errors.Is(err, store.ErrDuplicate) {
		return nil, ErrSocialLinkRefused
	} else if err != nil {
		return nil, err
	}

	if err := s.connectIdentity(ctx, user, provider, who, email, now); err != nil {
		return nil, err
	}

	s.record(ctx, user, user.Email, "user.registered", client, map[string]any{"provider": provider.Name})

	return s.startSocialSession(ctx, user, provider, flow, request, client)
}

func (s *Service) socialSSORequired(ctx context.Context, email string) error {
	err := s.ssoRequiredFor(ctx, email)

	if _, ok := errors.AsType[*SSORequired](err); ok {
		return ErrSocialSSORequired
	}

	return err
}

func (s *Service) socialRegistrationAllowed(ctx context.Context, request string) error {
	if request == "" {
		return nil
	}

	req, err := s.pending(ctx, request)
	if err != nil {
		return nil
	}

	if !req.Application.AllowRegistration || !req.Application.IsEnabled {
		return ErrSocialRegistrationClosed
	}

	return nil
}

func (s *Service) connectIdentity(
	ctx context.Context,
	user *model.User,
	provider *model.SocialProvider,
	who *socialIdentity,
	email string,
	now time.Time,
) error {
	identity := &model.SocialIdentity{
		UserID:      user.ID,
		ProviderID:  provider.ID,
		Subject:     who.Subject,
		Email:       email,
		LastLoginAt: &now,
	}

	return s.store.CreateSocialIdentity(ctx, identity)
}

func (s *Service) startSocialSession(
	ctx context.Context,
	user *model.User,
	provider *model.SocialProvider,
	flow *model.LoginFlow,
	request string,
	client Client,
) (*SignInResult, error) {
	result, err := s.startSession(ctx, user, flow, request, true, client, "user.login")
	if err != nil {
		return nil, err
	}

	s.log.Info("a user signed in with a social provider",
		"provider", provider.Slug, "user", user.ID)

	return result, nil
}

func claimString(claims map[string]any, path string) string {
	switch value := claimAt(claims, path).(type) {
	case string:
		return strings.TrimSpace(value)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case json.Number:
		return value.String()
	default:
		return ""
	}
}

func claimBool(claims map[string]any, path string) bool {
	switch value := claimAt(claims, path).(type) {
	case bool:
		return value
	case string:
		return value == "true"
	default:
		return false
	}
}

func claimAt(claims map[string]any, path string) any {
	if path == "" {
		return nil
	}

	var current any = claims
	for step := range strings.SplitSeq(path, ".") {
		switch node := current.(type) {
		case map[string]any:
			current = node[step]
		case []any:
			index, err := strconv.Atoi(step)
			if err != nil || index < 0 || index >= len(node) {
				return nil
			}
			current = node[index]
		default:
			return nil
		}
	}

	return current
}

func stringClaim(claims map[string]any, name string) string {
	return claimString(claims, name)
}

func numberClaim(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func audienceHas(aud any, clientID string) bool {
	switch value := aud.(type) {
	case string:
		return value == clientID
	case []any:
		for _, one := range value {
			if name, ok := one.(string); ok && name == clientID {
				return true
			}
		}
	}

	return false
}

func splitName(full string) (first, last string) {
	parts := strings.Fields(full)
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], ""
	default:
		return parts[0], strings.Join(parts[1:], " ")
	}
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
		description = reason.message
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
