package api

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// An address whose organisation requires its own connection is not let in
// by a provider's button either — not through an identity linked before the
// connection was required, not by linking one now, and not by making an
// account. The account's own address decides, not the provider's.
//
// The provider is this server, so once the domain is enforced nobody can
// sign in at the provider with a password: the browsers here are signed in
// there beforehand, and the provider's authorization endpoint lets a signed-in
// browser straight through.
func TestLiveSocialSignInHonoursRequiredSSO(t *testing.T) {
	f := newSocialFixture(t, nil)
	f.user(t, true)
	start := f.s.root + "/oauth2/social/" + f.slug + "/start"

	// A browser signed in at the provider, and nothing linked yet.
	unlinked := f.s.browser()
	if status := unlinked.account(http.MethodPost, "/login", map[string]any{
		"email": f.email, "password": f.password,
	}, nil); status != http.StatusOK {
		t.Fatalf("signing in at the provider beforehand = %d", status)
	}

	idp := newFakeOIDC(t)
	var created struct {
		Connection struct {
			ID string `json:"id"`
		} `json:"connection"`
	}
	f.super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Example", "issuer": idp.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret,
		"domains": []string{"example.com"}, "is_enabled": true, "enforce_domains": true,
	}, &created)
	enforce := func(on bool) {
		f.super.must(http.StatusOK, http.MethodPatch, "/sso-connections/"+created.Connection.ID, map[string]any{
			"enforce_domains": on,
		}, nil)
	}

	// through walks a browser that is already signed in at the provider: to
	// the provider, straight back with a code, and through the callback.
	through := func(t *testing.T, b *browser) *url.URL {
		t.Helper()

		atProvider := b.visit(start)
		callback := b.visit(atProvider.String())
		if !strings.Contains(callback.String(), "/oauth2/social/"+f.slug+"/callback") {
			t.Fatalf("the provider sent the browser to %s, want the callback", callback)
		}

		return b.visit(callback.String())
	}
	refused := func(t *testing.T, landed *url.URL) {
		t.Helper()

		if !strings.HasPrefix(landed.String(), testAccountURL+"/error") || landed.Query().Get("reason") != "social_sso_required" {
			t.Errorf("the sign-in landed on %s, want the error page saying social_sso_required", landed)
		}
	}

	// Linking by address is refused, and nothing is connected.
	refused(t, through(t, unlinked))
	if identities := f.identities(t); identities != 0 {
		t.Errorf("%d identities were connected, want none", identities)
	}

	// Linked while nothing was required, then required again: the identity
	// is no longer a way in.
	enforce(false)
	linked := f.s.browser()
	if landed := f.signInThere(t, linked, start); strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Fatalf("linking with the connection not required failed: %s", landed.Query().Get("reason"))
	}
	if identities := f.identities(t); identities != 1 {
		t.Fatalf("%d identities are connected, want one", identities)
	}
	enforce(true)

	refused(t, through(t, linked))
}
