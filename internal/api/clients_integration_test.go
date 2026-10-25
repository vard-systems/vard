package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"vard/internal/model"
)

// These tests hold the provider to what client libraries rely on: PKCE for
// every client without credentials.

// changeApplication changes an application in the store, past the panel's
// rules: for a row the panel could not make, as an older version or a
// hand-edited database might have left it.
func (s *liveServer) changeApplication(clientID string, change func(*model.Application)) {
	s.t.Helper()

	ctx := context.Background()
	app, err := s.store.ApplicationByClientID(ctx, clientID)
	if err != nil {
		s.t.Fatal(err)
	}
	change(app)
	if err := s.store.SaveApplication(ctx, app); err != nil {
		s.t.Fatal(err)
	}
}

// A public client has nothing but PKCE to tie a code to the app that asked
// for it, so it always uses PKCE — even if its row says otherwise. And a
// code asked for with a challenge is never exchanged without its verifier,
// whatever the application requires (the downgrade RFC 9700 §2.1.1 warns of).
func TestLivePKCECannotBeLeftOut(t *testing.T) {
	f := newOAuthFixture(t)

	spaID, _, _ := f.register(map[string]any{
		"name": "Storefront", "type": "spa",
		"grant_types":   []string{"authorization_code"},
		"redirect_uris": []string{"https://store.example.com/callback"},
		"scopes":        []string{"openid"},
	})
	f.s.changeApplication(spaID, func(app *model.Application) { app.RequirePKCE = false })

	location := f.s.browser().visit(f.s.root + "/oauth2/authorize?" + url.Values{
		"response_type": {"code"}, "client_id": {spaID}, "scope": {"openid"},
		"redirect_uri": {"https://store.example.com/callback"}, "state": {"s"},
	}.Encode())
	if location.Query().Get("error") != "invalid_request" || !strings.Contains(location.Query().Get("error_description"), "code_challenge") {
		t.Errorf("a public client's request without PKCE went to %s, want invalid_request about code_challenge", location)
	}

	// A confidential client that does not require PKCE, but asked with a
	// challenge, still has to prove the verifier.
	f.s.changeApplication(f.clientID, func(app *model.Application) { app.RequirePKCE = false })
	verifier, challenge := pkce(t)
	code := f.signIn(f.s.browser(), challenge).Query().Get("code")

	if downgraded := f.exchange(code, ""); downgraded.status != http.StatusBadRequest || downgraded.str("error") != "invalid_grant" {
		t.Errorf("exchange without the verifier = %d %v, want invalid_grant", downgraded.status, downgraded.body)
	}
	// The failed attempt spent the code, as any wrong answer does.
	if spent := f.exchange(code, verifier); spent.status != http.StatusBadRequest {
		t.Errorf("the code after a failed exchange = %d %v, want it spent", spent.status, spent.body)
	}
}
