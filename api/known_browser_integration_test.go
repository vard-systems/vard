package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/crewjam/saml"

	"vard/api/kit/session"
	"vard/tools/totp"
)

// knownCookiePath is where the known-browser cookie is kept over plain http.
const knownCookiePath = "/api/v1/account"

// knownTo says whether the browser's known-browser cookie is one this server
// takes for `email`: a wrong password from it is counted on a browser's own
// budget, not on the address's. The counts are emptied first, and after.
func (s *liveServer) knownTo(t *testing.T, b *browser, email string) bool {
	t.Helper()

	forget := func() {
		if err := s.db.Exec("DELETE FROM sign_in_throttles").Error; err != nil {
			t.Fatal(err)
		}
	}
	forget()
	defer forget()

	b.account(http.MethodPost, "/login", map[string]string{"email": email, "password": "not-the-password-1"}, nil)

	var browsers, addresses int64
	if err := s.db.Raw("SELECT count(*) FILTER (WHERE key LIKE 'browser:%'), count(*) FILTER (WHERE key LIKE 'email:%') FROM sign_in_throttles WHERE failures > 0").
		Row().Scan(&browsers, &addresses); err != nil {
		t.Fatal(err)
	}
	return browsers == 1 && addresses == 0
}

// Every way of signing in marks the browser as known to the account it
// signed in to — a password, a passkey alone, a social provider, single
// sign-on by OpenID Connect and by SAML — and a browser that has not is not.
func TestLiveEveryWayInMarksTheBrowserKnown(t *testing.T) {
	t.Run("a browser never signed in is not known", func(t *testing.T) {
		f := newOAuthFixture(t)
		if f.s.knownTo(t, f.s.browser(), adaEmail) {
			t.Error("a fresh browser was counted as known")
		}
	})

	t.Run("a password", func(t *testing.T) {
		f := newOAuthFixture(t)
		b := f.s.browser()
		_, challenge := pkce(t)
		f.signIn(b, challenge)
		if !f.s.knownTo(t, b, adaEmail) {
			t.Error("the browser is not known after a password sign-in")
		}
		if f.s.knownTo(t, b, "grace@example.com") {
			t.Error("the browser is known to an address it never signed in to")
		}
	})

	t.Run("a passkey alone", func(t *testing.T) {
		f := newOAuthFixture(t)
		f.withSteps("identifier", "password", "totp")
		owner := f.s.browser()
		_, challenge := pkce(t)
		f.signIn(owner, challenge)
		key := addPasskey(t, owner)

		b := f.s.browser()
		_, challenge = pkce(t)
		login := b.visit(f.authorizeURL(challenge, nil))
		var offer struct {
			Handle  string         `json:"handle"`
			Passkey map[string]any `json:"passkey"`
		}
		if status := b.account(http.MethodPost, "/login/passkey", map[string]any{"request": login.Query().Get("request")}, &offer); status != http.StatusOK {
			t.Fatalf("starting a passkey sign-in = %d", status)
		}
		if status := b.account(http.MethodPost, "/login/passkey/finish", map[string]any{
			"handle": offer.Handle, "response": json.RawMessage(key.get(t, offer.Passkey)),
		}, nil); status != http.StatusOK {
			t.Fatalf("finishing a passkey sign-in = %d", status)
		}
		if !f.s.knownTo(t, b, adaEmail) {
			t.Error("the browser is not known after a passkey sign-in")
		}
	})

	t.Run("a social provider", func(t *testing.T) {
		f := newSocialFixture(t, nil)
		f.user(t, true)
		b := f.s.browser()

		// This server is its own provider: the password at the provider marks
		// the browser first, and the callback marks it again, afresh.
		atProvider := b.visit(f.s.root + "/oauth2/social/" + f.slug + "/start")
		signInPage := b.visit(atProvider.String())
		var result struct {
			RedirectTo string `json:"redirect_to"`
		}
		b.account(http.MethodPost, "/login", map[string]any{
			"request": signInPage.Query().Get("request"), "email": f.email, "password": f.password,
		}, &result)
		before := b.cookieOf(session.KnownBrowserCookie, knownCookiePath)

		if landed := b.visit(result.RedirectTo); strings.HasPrefix(landed.String(), testAccountURL+"/error") {
			t.Fatalf("the social sign-in failed: %s", landed.Query().Get("error_description"))
		}
		if after := b.cookieOf(session.KnownBrowserCookie, knownCookiePath); after == "" || after == before {
			t.Error("the social callback did not mark the browser")
		}
		if !f.s.knownTo(t, b, f.email) {
			t.Error("the browser is not known after a social sign-in")
		}
	})

	t.Run("single sign-on by OpenID Connect", func(t *testing.T) {
		s := newLiveServer(t)
		super := s.superAdmin()
		idp := newFakeOIDC(t)
		super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
			"protocol": "oidc", "name": "Acme", "issuer": idp.server.URL,
			"client_id": fakeClientID, "client_secret": fakeClientSecret,
			"domains": []string{"acme.test"}, "is_enabled": true,
		}, nil)
		idp.signs(map[string]any{"sub": "okta|ada", "email": "ada@acme.test", "email_verified": true, "given_name": "Ada"})

		b := s.browser()
		if reason, failed := idp.signInOIDC(b, "acme").failed(); failed {
			t.Fatalf("the sign-in failed: %s", reason)
		}
		if !s.knownTo(t, b, "ada@acme.test") {
			t.Error("the browser is not known after an OpenID Connect single sign-on")
		}
	})

	t.Run("single sign-on by SAML", func(t *testing.T) {
		s := newLiveServer(t)
		super := s.superAdmin()
		idp := newFakeSAML(t)
		super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
			"protocol": "saml", "name": "Globex", "metadata": idp.metadata(t),
			"domains": []string{"globex.test"}, "is_enabled": true, "name_id_format": "persistent",
		}, nil)
		idp.mu.Lock()
		idp.person = &saml.Session{
			ID: "session-1", CreateTime: time.Now(), ExpireTime: time.Now().Add(time.Hour),
			NameID: "globex-7", UserEmail: "hedy@globex.test", UserGivenName: "Hedy",
		}
		idp.mu.Unlock()

		b := s.browser()
		result, _ := idp.signInSAML(t, b, "globex")
		if reason, failed := result.failed(); failed {
			t.Fatalf("the sign-in failed: %s", reason)
		}
		if !s.knownTo(t, b, "hedy@globex.test") {
			t.Error("the browser is not known after a SAML single sign-on")
		}
	})
}

// A known browser has a budget of its own, not none: a second factor guessed
// from it is counted there, as a password is, and past the free failures the
// right code waits too. Being known never skips the count.
func TestLiveAKnownBrowserStillCountsItsSecondFactorGuesses(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")

	b := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(b, challenge)
	secret, _ := addAuthenticator(t, b)

	// Every sign-in from it now asks for the code.
	if !f.s.knownTo(t, b, adaEmail) {
		t.Fatal("the browser is not known before the guesses")
	}

	// Three rounds of the right password and a wrong code: the codes are
	// counted with the passwords on the browser's budget, so the fifth
	// attempt (the third password) makes it wait. Were the codes skipped, the
	// three passwords alone would leave room for more.
	for i := range 3 {
		_, challenge := pkce(t)
		login := b.visit(f.authorizeURL(challenge, nil))
		var out struct {
			SecondFactor bool `json:"second_factor"`
		}
		if status := b.account(http.MethodPost, "/login", map[string]string{
			"request": login.Query().Get("request"), "email": adaEmail, "password": adaPassword,
		}, &out); status != http.StatusOK || !out.SecondFactor {
			t.Fatalf("password %d = %d %+v, want a second factor asked for", i+1, status, out)
		}
		answerSecondFactor(b, map[string]any{"code": wrongCodeFor(t, secret)})
	}

	// The waiting sign-in of the last round, with the right code: it waits.
	if status, _ := answerSecondFactor(b, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}); status == http.StatusOK {
		t.Error("the right code after the browser's budget ran out = 200, want it waiting")
	}

	var browserFailures, addressFailures int
	if err := f.s.db.Raw("SELECT COALESCE(MAX(failures) FILTER (WHERE key LIKE 'browser:%'), 0), COALESCE(MAX(failures) FILTER (WHERE key LIKE 'email:%'), 0) FROM sign_in_throttles").
		Row().Scan(&browserFailures, &addressFailures); err != nil {
		t.Fatal(err)
	}
	if browserFailures < 5 || addressFailures != 0 {
		t.Errorf("counts: the browser's %d, the address's %d; want every password and code on the browser's", browserFailures, addressFailures)
	}
}
