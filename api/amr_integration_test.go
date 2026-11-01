package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"vard/tools/totp"
)

// amrOf is the amr claim of a token answer's ID token.
func amrOf(t *testing.T, tokens tokenResult) []string {
	t.Helper()

	parts := strings.Split(tokens.str("id_token"), ".")
	if len(parts) != 3 {
		t.Fatalf("no ID token in %v", tokens.body)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims struct {
		AMR []string `json:"amr"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	return claims.AMR
}

// An application can tell how the person signed in (RFC 8176): a password
// alone, or a password and a second factor, which is multi-factor — and a
// refresh says what the sign-in it came from did.
func TestLiveTheIDTokenSaysHowSomeoneSignedIn(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "social", "totp")

	owner := f.s.browser()
	verifier, challenge := pkce(t)
	first := f.exchange(f.signIn(owner, challenge).Query().Get("code"), verifier)
	if got := amrOf(t, first); !slices.Equal(got, []string{"pwd"}) {
		t.Errorf("a password alone: amr = %v, want [pwd]", got)
	}

	secret, _ := addAuthenticator(t, owner)

	b := f.s.browser()
	verifier, challenge = pkce(t)
	login := b.visit(f.authorizeURL(challenge, nil))
	b.account(http.MethodPost, "/login", map[string]any{
		"request": login.Query().Get("request"), "email": adaEmail, "password": adaPassword,
	}, nil)
	status, to := answerSecondFactor(b, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))})
	if status != http.StatusOK {
		t.Fatalf("the second factor = %d %s", status, to)
	}
	back, _ := url.Parse(to)

	tokens := f.exchange(back.Query().Get("code"), verifier)
	want := []string{"pwd", "otp", "mfa"}
	if got := amrOf(t, tokens); !slices.Equal(got, want) {
		t.Errorf("a password and an authenticator: amr = %v, want %v", got, want)
	}

	refreshed := f.refresh(tokens.str("refresh_token"))
	if got := amrOf(t, refreshed); !slices.Equal(got, want) {
		t.Errorf("after a refresh: amr = %v, want %v", got, want)
	}
}
