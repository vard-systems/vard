package oidc

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"vard/internal/model"
)

// unsignedIDToken is a JWT with these claims and nothing else worth reading:
// idTokenClaims does not check the signature, so a header and a payload are
// all it takes.
func unsignedIDToken(t *testing.T, claims map[string]any) string {
	t.Helper()

	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}

	b64 := base64.RawURLEncoding

	return b64.EncodeToString([]byte(`{"alg":"none"}`)) + "." + b64.EncodeToString(payload) + ".not-checked"
}

// An id_token the identity is read out of is held to the client it was issued
// for, to the issuer its kind is known to have, to the sign-in it was asked
// for (its nonce), and to the clock. The token
// endpoint answering over TLS says the answer came from the provider; it does
// not say the token inside it is one of theirs.
func TestIDTokenClaimsHoldsATokenToItsProvider(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	service := &Service{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		now: func() time.Time { return now },
	}

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
			_, err := service.idTokenClaims(tt.provider, unsignedIDToken(t, tt.claims), nonce)
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
