package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"vard/internal/domain/model"
	"vard/internal/jose"
)

// unsignedIDToken is a JWT with these claims and nothing else worth reading,
// as a provider that publishes no keys is trusted to send: a header and a
// payload are all it takes.
func unsignedIDToken(t *testing.T, claims map[string]any) string {
	t.Helper()

	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}

	b64 := base64.RawURLEncoding

	return b64.EncodeToString([]byte(`{"alg":"none"}`)) + "." + b64.EncodeToString(payload) + ".not-checked"
}

func signedIDToken(t *testing.T, key jose.Key, claims map[string]any) string {
	t.Helper()

	token, err := jose.Sign(key, "JWT", claims)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// A provider that publishes its keys has its id_tokens' signatures checked
// against them, so a token from anyone else is refused even when it comes
// back through the token endpoint. A key the provider has rotated in since
// the keys were fetched makes them be fetched again, once; a key already
// held does not.
func TestIDTokenClaimsChecksThePublishedSignature(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	const nonce = "this-sign-in"
	apple := &model.SocialProvider{Kind: model.SocialApple, Slug: "apple", ClientID: "com.example.app"}
	claims := map[string]any{
		"iss": "https://appleid.apple.com", "aud": "com.example.app", "sub": "000123",
		"exp": now.Add(time.Hour).Unix(), "nonce": nonce,
	}

	current, rotated := signingKey(t, "apple-1"), signingKey(t, "apple-2")
	impostor := signingKey(t, "apple-1")

	tests := []struct {
		name    string
		token   string
		want    error
		fetches int
	}{
		{name: "signed with the published key", token: signedIDToken(t, current, claims), fetches: 1},
		{name: "signed by somebody else under the published kid", token: signedIDToken(t, impostor, claims), want: ErrSocialUpstream, fetches: 1},
		{name: "not signed at all", token: unsignedIDToken(t, claims), want: ErrSocialUpstream, fetches: 2},
		{name: "signed with a key published since", token: signedIDToken(t, rotated, claims), fetches: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			published := &publishedKeys{keys: []jose.Key{current}}
			service := serviceWithKeys(published, &clock{at: now})
			if _, err := service.providerKeys(context.Background(), apple.Spec().JWKSURL, false); err != nil {
				t.Fatal(err)
			}
			published.publish(current, rotated)

			got, err := service.idTokenClaims(context.Background(), apple, tt.token, nonce)
			if !errors.Is(err, tt.want) {
				t.Fatalf("idTokenClaims = %v, want %v", err, tt.want)
			}
			if tt.want == nil && claimString(got, "sub") != "000123" {
				t.Errorf("claims = %v", got)
			}
			if fetches := published.count(); fetches != tt.fetches {
				t.Errorf("fetches = %d, want %d", fetches, tt.fetches)
			}
		})
	}
}

// An id_token the identity is read out of is held to the client it was issued
// for, to the issuer its kind is known to have, to the sign-in it was asked
// for (its nonce), and to the clock. The token
// endpoint answering over TLS says the answer came from the provider; it does
// not say the token inside it is one of theirs.
func TestIDTokenClaimsHoldsATokenToItsProvider(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	appleKey := signingKey(t, "apple-1")
	service := serviceWithKeys(&publishedKeys{keys: []jose.Key{appleKey}}, &clock{at: now})

	apple := &model.SocialProvider{Kind: model.SocialApple, Slug: "apple", ClientID: "com.example.app"}
	inHouse := &model.SocialProvider{Kind: model.SocialOIDC, Slug: "in-house", ClientID: "in-house-app"}

	const nonce = "this-sign-in"
	claims := func(issuer, audience string, expires time.Time) map[string]any {
		return map[string]any{"iss": issuer, "aud": audience, "sub": "000123", "exp": expires.Unix(), "nonce": nonce}
	}
	with := func(base map[string]any, key string, value any) map[string]any {
		changed := map[string]any{}
		for k, v := range base {
			changed[k] = v
		}
		if value == nil {
			delete(changed, key)
		} else {
			changed[key] = value
		}
		return changed
	}
	good := claims("https://appleid.apple.com", "com.example.app", now.Add(time.Hour))

	tests := []struct {
		name     string
		provider *model.SocialProvider
		claims   map[string]any
		want     error
	}{
		{
			name:     "the provider's own token",
			provider: apple,
			claims:   claims("https://appleid.apple.com", "com.example.app", now.Add(time.Hour)),
		},
		{
			name:     "another issuer, with this client's audience",
			provider: apple,
			claims:   claims("https://not-apple.example", "com.example.app", now.Add(time.Hour)),
			want:     ErrSocialUpstream,
		},
		{
			name:     "no issuer at all",
			provider: apple,
			claims:   with(good, "iss", nil),
			want:     ErrSocialUpstream,
		},
		{
			name:     "no expiry at all",
			provider: apple,
			claims:   with(good, "exp", nil),
			want:     ErrSocialUpstream,
		},
		{
			name:     "issued in the future",
			provider: apple,
			claims:   with(good, "iat", now.Add(time.Hour).Unix()),
			want:     ErrSocialUpstream,
		},
		{
			name:     "authorised for another client",
			provider: apple,
			claims:   with(with(good, "aud", []string{"com.example.app", "com.other.app"}), "azp", "com.other.app"),
			want:     ErrSocialUpstream,
		},
		{
			name:     "another sign-in's nonce",
			provider: apple,
			claims:   with(good, "nonce", "another-sign-in"),
			want:     ErrSocialUpstream,
		},
		{
			name:     "no nonce, though one was sent",
			provider: apple,
			claims:   with(good, "nonce", nil),
			want:     ErrSocialUpstream,
		},
		{
			name:     "another client's token",
			provider: apple,
			claims:   claims("https://appleid.apple.com", "com.somebody-else.app", now.Add(time.Hour)),
			want:     ErrSocialUpstream,
		},
		{
			name:     "a token that has expired",
			provider: apple,
			claims:   claims("https://appleid.apple.com", "com.example.app", now.Add(-idTokenLeeway-time.Second)),
			want:     ErrSocialExpired,
		},
		{
			name:     "a kind whose issuer nobody knows in advance",
			provider: inHouse,
			claims:   claims("https://whatever.example", "in-house-app", now.Add(time.Hour)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := unsignedIDToken(t, tt.claims)
			if tt.provider.Spec().JWKSURL != "" {
				token = signedIDToken(t, appleKey, tt.claims)
			}
			_, err := service.idTokenClaims(context.Background(), tt.provider, token, nonce)
			if !errors.Is(err, tt.want) {
				t.Errorf("idTokenClaims = %v, want %v", err, tt.want)
			}
		})
	}
}

// After a provider sign-in the browser is sent to `next`, which arrives as a
// query parameter anyone can write: it may only ever name a page of the
// account app, never another site, by any of the spellings browsers accept.
func TestAccountLandingStaysOnTheAccountApp(t *testing.T) {
	s := &Service{accountURL: "https://id.example"}

	tests := []struct {
		name string
		next string
		want string
	}{
		{name: "a page of the app", next: "/sessions", want: "https://id.example/sessions"},
		{name: "a page with a query", next: "/sessions?tab=all", want: "https://id.example/sessions?tab=all"},
		{name: "nothing", next: "", want: "https://id.example/"},
		{name: "a relative path", next: "sessions", want: "https://id.example/"},
		{name: "another site, protocol-relative", next: "//evil.example", want: "https://id.example/"},
		{name: "another site, with a backslash", next: `/\evil.example`, want: "https://id.example/"},
		{name: "another site, absolute", next: "https://evil.example/", want: "https://id.example/"},
		{name: "a script address", next: "javascript:alert(1)", want: "https://id.example/"},
		{name: "a tab browsers would strip", next: "/\t/evil.example", want: "https://id.example/"},
		{name: "a newline browsers would strip", next: "/\n/evil.example", want: "https://id.example/"},
		{name: "a delete character", next: "/\x7f/evil.example", want: "https://id.example/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.accountLanding(tt.next); got != tt.want {
				t.Errorf("accountLanding(%q) = %q, want %q", tt.next, got, tt.want)
			}
		})
	}
}
