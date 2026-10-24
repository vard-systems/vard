package oidc

import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"loginer/internal/jose"
	"loginer/internal/model"
)

// Talking to a social provider: trading the code for tokens, proving who the
// person is from its ID token or profile endpoint, and reading claims out of
// what it sent.

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
		AccessToken: claimString(answer, "access_token"),
		IDToken:     claimString(answer, "id_token"),
		Raw:         answer,
	}

	if token.AccessToken == "" && token.IDToken == "" {
		s.log.Error("a social provider gave no token",
			"provider", provider.Slug, "error", claimString(answer, "error"))
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
		identity.Email = claimString(token.Raw, "email")
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
	if want := provider.Spec().IDTokenIssuer; want != "" && claimString(claims, "iss") != want {
		s.log.Error("a social provider's id_token named another issuer",
			"provider", provider.Slug, "issuer", claimString(claims, "iss"), "want", want)
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
