package api

import (
	"net/http"
	"testing"
)

// The switches a login flow carries decide what a sign-in allows, and the
// server holds them whatever a page offers: a closed flow lets nobody in by
// any road, and a flow that does not allow an address change refuses one.
func TestLiveLoginSwitchesDecideWhatIsAllowed(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	flow := defaultFlow(t, super)
	change := func(body map[string]any) {
		super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+flow, body, nil)
	}

	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "ada@example.com", "password": "ada-password-1", "confirm_password": "ada-password-1",
	}, nil)

	login := func(b *browser, remember bool) (int, problemBody) {
		var out problemBody
		status := b.account(http.MethodPost, "/login", map[string]any{
			"email": "ada@example.com", "password": "ada-password-1", "remember": remember,
		}, &out)

		return status, out
	}

	// Closed: nobody gets in, and the pages are told so rather than being
	// left to ask and be refused.
	change(map[string]any{"allow_sign_in": false})

	if status, out := login(s.browser(), false); status != http.StatusForbidden || out.Code != "sign_in_closed" {
		t.Errorf("signing in through a closed flow = %d %+v, want sign_in_closed", status, out)
	}

	var options struct {
		Login struct {
			AllowSignIn      bool `json:"allow_sign_in"`
			AllowRememberMe  bool `json:"allow_remember_me"`
			AllowEmailChange bool `json:"allow_email_change"`
		} `json:"login"`
	}
	s.public("/login-options", &options)
	if options.Login.AllowSignIn {
		t.Error("the sign-in pages were told sign-ins are open while the flow is closed")
	}

	// Open again, and the switches the pages read follow.
	change(map[string]any{"allow_sign_in": true, "allow_remember_me": true, "allow_email_change": true})

	s.public("/login-options", &options)
	if !options.Login.AllowSignIn || !options.Login.AllowRememberMe || !options.Login.AllowEmailChange {
		t.Errorf("login options = %+v, want every switch on", options.Login)
	}

	// Remembered or not, the session is the same; what differs is how long
	// the browser is told to hold its cookie.
	remembered := s.browser()
	if status, out := login(remembered, true); status != http.StatusOK {
		t.Fatalf("signing in = %d %+v", status, out)
	}

	// A session alone cannot move the account: a new address plus a password
	// reset would hand it to whoever holds the cookie.
	for _, attempt := range []struct {
		name string
		body map[string]string
	}{
		{"no password", map[string]string{"email": "ada.new@example.com"}},
		{"a wrong password", map[string]string{"email": "ada.new@example.com", "current_password": "not-adas-password"}},
	} {
		if status := remembered.account(http.MethodPost, "/email", attempt.body, nil); status != http.StatusBadRequest {
			t.Errorf("asking to change the address with %s = %d, want 400", attempt.name, status)
		}
	}

	// Moving to another address: the link goes to the new one, and nothing
	// changes until it is used.
	if status := remembered.account(http.MethodPost, "/email", map[string]string{
		"email": "ada.new@example.com", "current_password": "ada-password-1",
	}, nil); status != http.StatusAccepted {
		t.Fatalf("asking to change the address = %d, want 202", status)
	}

	sent := s.mail.wait(t, "ada.new@example.com")
	token := tokenIn(t, sent.Body, "/verify-email")

	var before struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	remembered.account(http.MethodGet, "/me", nil, &before)
	if before.User.Email != "ada@example.com" {
		t.Errorf("the address moved before the link was used: %s", before.User.Email)
	}

	if status := remembered.account(http.MethodPost, "/verify-email", map[string]string{"token": token}, nil); status != http.StatusOK {
		t.Fatalf("using the link = %d, want 200", status)
	}

	var after struct {
		User struct {
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
		} `json:"user"`
	}
	remembered.account(http.MethodGet, "/me", nil, &after)
	if after.User.Email != "ada.new@example.com" || !after.User.EmailVerified {
		t.Errorf("after the link, the account is %+v, want the new address, verified", after.User)
	}

	// The flow can take the offer away again, and the server refuses it
	// whatever the page shows.
	change(map[string]any{"allow_email_change": false})

	var refused problemBody
	if status := remembered.account(http.MethodPost, "/email", map[string]string{
		"email": "ada.third@example.com", "current_password": "ada-password-1",
	}, &refused); status != http.StatusForbidden || refused.Code != "email_change_not_offered" {
		t.Errorf("changing the address where the flow forbids it = %d %+v", status, refused)
	}
}

// A flow that confirms new accounts sends the link and lets them in; one that
// requires a verified address holds them back instead. They are two switches
// and two different outcomes.
func TestLiveVerifyEmailOnRegister(t *testing.T) {
	f := newOAuthFixture(t)

	flow := defaultFlow(t, f.super)
	f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+flow, map[string]any{
		"verify_email_on_register": true, "require_verified_email": false,
	}, nil)
	// An application update is validated as a whole record, so the fields the
	// fixture registered this one with come along unchanged; only
	// allow_registration is being moved.
	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+f.appID, map[string]any{
		"name": "Shop", "type": "web",
		"grant_types":        []string{"authorization_code", "refresh_token"},
		"redirect_uris":      []string{shopRedirect},
		"scopes":             []string{"openid", "profile", "email", "offline_access", "roles"},
		"allow_registration": true,
	}, nil)

	b := f.s.browser()
	_, challenge := pkce(t)
	login := b.visit(f.authorizeURL(challenge, nil))

	if status := b.account(http.MethodPost, "/register", map[string]any{
		"request":  login.Query().Get("request"),
		"email":    "grace@example.com",
		"password": "grace-password-1",
	}, nil); status != http.StatusOK {
		t.Fatalf("registering = %d, want 200", status)
	}

	// Signed in, and sent a link — confirming the address does not hold the
	// account back; requiring a verified one is what does that.
	if status := b.account(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Error("a new account was not signed in, though the flow only confirms the address")
	}

	// linkIn fails the test when the message carries none.
	linkIn(t, f.s.mail.wait(t, "grace@example.com").Body, "/verify-email")
}
