package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"vard/core/model"
	"vard/tools/jose"
)

type oidcDiscovery struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserInfoEndpoint      string   `json:"userinfo_endpoint"`
	JWKSURI               string   `json:"jwks_uri"`
	TokenAuthMethods      []string `json:"token_endpoint_auth_methods_supported"`
	ScopesSupported       []string `json:"scopes_supported"`
}

// ssoCache holds what identity providers publish about themselves, for an
// hour, so a sign-in does not fetch their discovery document and keys again.
// forced is when each key set was last fetched again because a token did not
// verify, which anyone can cause by sending a bad one.
type ssoCache struct {
	mu        sync.Mutex
	discovery map[string]cached[*oidcDiscovery]
	keys      map[string]cached[[]jose.Key]
	forced    map[string]time.Time
}

type cached[T any] struct {
	value   T
	fetched time.Time
}

const (
	ssoCacheLifetime = time.Hour
	// keyRefetchEvery is how often a token that does not verify may make
	// the server fetch the provider's keys again: often enough to follow a
	// rotation, too seldom for forged tokens to make it fetch on each one.
	keyRefetchEvery = time.Minute
)

func newSSOCache() *ssoCache {
	return &ssoCache{
		discovery: map[string]cached[*oidcDiscovery]{},
		keys:      map[string]cached[[]jose.Key]{},
		forced:    map[string]time.Time{},
	}
}

func (s *Service) Discover(ctx context.Context, issuer string) (*oidcDiscovery, error) {
	issuer = strings.TrimRight(issuer, "/")

	s.ssoCache.mu.Lock()
	if hit, ok := s.ssoCache.discovery[issuer]; ok && s.now().Sub(hit.fetched) < ssoCacheLifetime {
		s.ssoCache.mu.Unlock()
		return hit.value, nil
	}
	s.ssoCache.mu.Unlock()

	var document oidcDiscovery
	if err := s.getJSON(ctx, issuer+"/.well-known/openid-configuration", &document); err != nil {
		return nil, err
	}

	// The document must be the issuer's own (OpenID Connect Discovery 4.3), or
	// another issuer's tokens would be believed.
	switch {
	case strings.TrimRight(document.Issuer, "/") != issuer:
		return nil, fmt.Errorf("the discovery document is for %q, not %q", document.Issuer, issuer)
	case document.AuthorizationEndpoint == "" || document.TokenEndpoint == "" || document.JWKSURI == "":
		return nil, fmt.Errorf("the discovery document is missing an endpoint")
	}

	s.ssoCache.mu.Lock()
	s.ssoCache.discovery[issuer] = cached[*oidcDiscovery]{value: &document, fetched: s.now()}
	s.ssoCache.mu.Unlock()

	return &document, nil
}

func (s *Service) startOIDC(ctx context.Context, connection *model.SSOConnection, login *model.SSOLogin, state, loginHint string) (string, error) {
	document, err := s.Discover(ctx, connection.Issuer)
	if err != nil {
		return "", upstream("discovery", err)
	}

	verifier, _, err := model.NewSecret()
	if err != nil {
		return "", err
	}
	nonce, _, err := model.NewSecret()
	if err != nil {
		return "", err
	}
	login.Verifier, login.Nonce = verifier, nonce

	values := url.Values{
		"response_type":         {"code"},
		"client_id":             {connection.ClientID},
		"redirect_uri":          {connection.CallbackURL(s.issuer)},
		"scope":                 {strings.Join(connection.AskedScopes(), " ")},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {challengeFor(verifier)},
		"code_challenge_method": {model.PKCES256},
	}
	if loginHint != "" {
		values.Set("login_hint", loginHint)
	}

	return withQuery(document.AuthorizationEndpoint, values), nil
}

func (s *Service) completeOIDC(ctx context.Context, connection *model.SSOConnection, login *model.SSOLogin, code string) (*ssoPerson, error) {
	document, err := s.Discover(ctx, connection.Issuer)
	if err != nil {
		return nil, upstream("discovery", err)
	}

	secret := ""
	if len(connection.ClientSecret) > 0 {
		plain, err := s.sealer.OpenFor(connection.ClientSecretWhere(), connection.ClientSecret)
		if err != nil {
			return nil, err
		}
		secret = string(plain)
	}

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {connection.CallbackURL(s.issuer)},
		"code_verifier": {login.Verifier},
	}

	basic := len(document.TokenAuthMethods) == 0 || slices.Contains(document.TokenAuthMethods, "client_secret_basic")
	if !basic {
		form.Set("client_id", connection.ClientID)
		form.Set("client_secret", secret)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, document.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	if basic {
		request.SetBasicAuth(url.QueryEscape(connection.ClientID), url.QueryEscape(secret))
	}

	var token struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := s.doJSON(request, &token); err != nil {
		return nil, upstream("token", err)
	}

	claims, err := s.verifyIDToken(ctx, connection, document, login, token.IDToken)
	if err != nil {
		return nil, upstream("id_token", err)
	}

	// What the sign-in proved is the signed id_token's to say: read before
	// userinfo, unsigned, fills in what the token left out.
	proved := idTokenAMR(claims)

	if claimFirst(claims, connection.Attribute("email")) == "" && document.UserInfoEndpoint != "" && token.AccessToken != "" {
		if extra, err := s.oidcUserInfo(ctx, document.UserInfoEndpoint, token.AccessToken); err == nil &&
			claimString(extra, "sub") == claimString(claims, "sub") {
			for name, value := range extra {
				if _, has := claims[name]; !has {
					claims[name] = value
				}
			}
		}
	}

	// A provider that says outright it has not verified the address is not
	// vouching for it.
	email := claimFirst(claims, connection.Attribute("email"))
	if said, verified := emailVerified(claims["email_verified"]); said && !verified && email != "" {
		return nil, refused(ErrSSONoEmail, "the provider says it has not verified %s (email_verified is false)", email)
	}

	return &ssoPerson{
		Subject:   claimString(claims, "sub"),
		Email:     email,
		FirstName: claimFirst(claims, connection.Attribute("first_name")),
		LastName:  claimFirst(claims, connection.Attribute("last_name")),
		Groups:    claimList(claims, connection.Attribute("groups")),
		Proved:    proved,
		Sent:      slices.Sorted(maps.Keys(claims)),
	}, nil
}

// idTokenAMR is an id_token's amr: an array of strings (RFC 8176), and
// nothing else read as one.
func idTokenAMR(claims map[string]any) []string {
	values, _ := claims["amr"].([]any)
	var amr []string
	for _, value := range values {
		if text, ok := value.(string); ok && text != "" {
			amr = append(amr, text)
		}
	}
	return amr
}

// emailVerified reads email_verified as a boolean or, as Cognito and others
// send it, a string: "false" must not read as unsaid.
func emailVerified(value any) (said, verified bool) {
	switch v := value.(type) {
	case bool:
		return true, v
	case string:
		return true, strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false, false
	}
}

const idTokenLeeway = time.Minute

// verifyIDToken checks an id_token is the provider's, for this client, in
// time, and for this sign-in (OpenID Connect Core section 3.1.3.7).
func (s *Service) verifyIDToken(
	ctx context.Context,
	connection *model.SSOConnection,
	document *oidcDiscovery,
	login *model.SSOLogin,
	idToken string,
) (map[string]any, error) {
	if idToken == "" {
		return nil, fmt.Errorf("no id_token")
	}

	var claims map[string]any
	if err := s.verifyPublished(ctx, document.JWKSURI, idToken, &claims); err != nil {
		return nil, err
	}

	now := s.now()

	switch {
	case claimString(claims, "iss") != document.Issuer:
		return nil, fmt.Errorf("issued by %q", claimString(claims, "iss"))
	case !audienceHas(claims["aud"], connection.ClientID):
		return nil, fmt.Errorf("for another client")
	case claimString(claims, "nonce") != login.Nonce:
		return nil, fmt.Errorf("for another sign-in")
	}

	if azp := claimString(claims, "azp"); azp != "" && azp != connection.ClientID {
		return nil, fmt.Errorf("authorised for another client")
	}

	exp, ok := numberClaim(claims["exp"])
	if !ok || now.After(time.Unix(int64(exp), 0).Add(idTokenLeeway)) {
		return nil, fmt.Errorf("expired")
	}
	if iat, ok := numberClaim(claims["iat"]); ok && time.Unix(int64(iat), 0).After(now.Add(idTokenLeeway)) {
		return nil, fmt.Errorf("issued in the future")
	}

	return claims, nil
}

// verifyPublished checks a token's signature against the keys a provider
// publishes at jwksURI and decodes its claims. A token naming a key not held,
// or naming none, may be signed with one the provider has just rotated in,
// so the keys are fetched again, at most once a keyRefetchEvery. One naming a
// key held is simply refused: fetching again cannot help it, and it would
// spend the refetch a real rotation may need.
func (s *Service) verifyPublished(ctx context.Context, jwksURI, token string, claims any) error {
	keys, err := s.providerKeys(ctx, jwksURI, false)
	if err != nil {
		return err
	}

	// kid is optional in a JWS: a token without one may be signed by any of
	// the provider's keys of its algorithm, and one with a kid only by that.
	candidates := func(keys []jose.Key) func(jose.Header) []jose.Key {
		return func(header jose.Header) []jose.Key {
			if header.KeyID == "" {
				return keys
			}
			for _, key := range keys {
				if key.ID == header.KeyID {
					return []jose.Key{key}
				}
			}
			return nil
		}
	}

	read, held := false, false
	first := func(header jose.Header) []jose.Key {
		found := candidates(keys)(header)
		read, held = true, header.KeyID != "" && len(found) > 0
		return found
	}
	if _, err := jose.VerifyAmong(token, first, claims); err == nil {
		return nil
	} else if !read || held {
		return fmt.Errorf("signature: %w", err)
	}
	if keys, err = s.providerKeys(ctx, jwksURI, true); err != nil {
		return err
	}
	if _, err := jose.VerifyAmong(token, candidates(keys), claims); err != nil {
		return fmt.Errorf("signature: %w", err)
	}

	return nil
}

// providerKeys is the key set published at jwksURI, from the cache while it
// is fresh. refresh asks for it to be fetched again, which happens at most
// once a keyRefetchEvery per address; in between, the keys held are the
// answer.
func (s *Service) providerKeys(ctx context.Context, jwksURI string, refresh bool) ([]jose.Key, error) {
	s.ssoCache.mu.Lock()
	hit, ok := s.ssoCache.keys[jwksURI]
	switch {
	case ok && !refresh && s.now().Sub(hit.fetched) < ssoCacheLifetime:
		s.ssoCache.mu.Unlock()
		return hit.value, nil
	case refresh && s.now().Sub(s.ssoCache.forced[jwksURI]) < keyRefetchEvery:
		s.ssoCache.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("the provider's keys were fetched again less than %s ago", keyRefetchEvery)
		}
		return hit.value, nil
	case refresh:
		// Noted before fetching, so requests arriving while it is under way
		// do not fetch too, and a failed fetch counts as much as one that
		// worked.
		s.ssoCache.forced[jwksURI] = s.now()
	}
	s.ssoCache.mu.Unlock()

	var set struct {
		Keys []jose.JWK `json:"keys"`
	}
	if err := s.getJSON(ctx, jwksURI, &set); err != nil {
		return nil, err
	}

	// A list, not a map by kid: keys published without one must not
	// overwrite each other.
	var keys []jose.Key
	for _, jwk := range set.Keys {
		if jwk.Use != "" && jwk.Use != "sig" {
			continue
		}
		// A key of a kind this server cannot verify with is skipped, not an
		// error: the provider may publish others beside the one it uses.
		if key, err := jose.ParseJWK(jwk); err == nil {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("the provider publishes no key this server can verify with")
	}

	s.ssoCache.mu.Lock()
	s.ssoCache.keys[jwksURI] = cached[[]jose.Key]{value: keys, fetched: s.now()}
	s.ssoCache.mu.Unlock()

	return keys, nil
}

func (s *Service) oidcUserInfo(ctx context.Context, endpoint, accessToken string) (map[string]any, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")

	var claims map[string]any
	return claims, s.doJSON(request, &claims)
}

func (s *Service) getJSON(ctx context.Context, address string, into any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")

	return s.doJSON(request, into)
}

func (s *Service) doJSON(request *http.Request, into any) error {
	ctx, cancel := context.WithTimeout(request.Context(), socialTimeout)
	defer cancel()

	response, err := s.social.Do(request.WithContext(ctx))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %d: %.200s", request.URL.Host, response.StatusCode, body)
	}

	return json.Unmarshal(body, into)
}

func claimFirst(claims map[string]any, names []string) string {
	for _, name := range names {
		if value := claimString(claims, name); value != "" {
			return value
		}
	}

	return ""
}

func claimList(claims map[string]any, names []string) []string {
	for _, name := range names {
		switch value := claimAt(claims, name).(type) {
		case []any:
			var out []string
			for _, item := range value {
				if text, ok := item.(string); ok && text != "" {
					out = append(out, text)
				}
			}
			return out
		case string:
			if value != "" {
				return []string{value}
			}
		}
	}

	return nil
}
