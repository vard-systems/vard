package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"vard/internal/domain/totp"
	"vard/internal/infrastructure/mail"
	"vard/internal/interfaces/api/kit/session"
)

// Setting up a second factor while signing in to an application whose flow
// requires one, and the notice every new factor sends.

const noticeSubject = "Two-step verification was turned on"

// enrolOffer is what the second-factor page is told.
type enrolOffer struct {
	SecondFactor struct {
		Enrol   bool     `json:"enrol"`
		StepUp  bool     `json:"step_up"`
		Methods []string `json:"methods"`
		// Application is named when no sign-in request does: a device.
		Application *struct {
			Name string `json:"name"`
		} `json:"application"`
	} `json:"second_factor"`
}

// enrolled is the answer to setting a factor up while signing in.
type enrolled struct {
	RedirectTo    string   `json:"redirect_to"`
	RecoveryCodes []string `json:"recovery_codes"`
	Code          string   `json:"code"`
}

// accountWith is account with headers of its own, returning the response's
// headers too.
func (b *browser) accountWith(method, path string, body any, header http.Header, out any) (int, http.Header) {
	b.t.Helper()

	encoded, _ := json.Marshal(body)
	req, err := http.NewRequest(method, b.s.root+"/api/v1/account"+path, bytes.NewReader(encoded))
	if err != nil {
		b.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, values := range header {
		req.Header[key] = values
	}

	res, err := b.http.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	if out != nil {
		_ = json.Unmarshal(raw, out)
	}

	return res.StatusCode, res.Header
}

// signInAs signs b in with a password, for the sign-in under way named by
// request ("" for none), and decodes the answer.
func (b *browser) signInAs(email, password, request string, remember bool, out any) int {
	b.t.Helper()
	return b.account(http.MethodPost, "/login", map[string]any{
		"request": request, "email": email, "password": password, "remember": remember,
	}, out)
}

// makeUser makes an active account with a password.
func (f *oauthFixture) makeUser(email, password string, verified bool) {
	f.s.t.Helper()
	f.super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": email, "first_name": "Test", "last_name": "Person",
		"password": password, "confirm_password": password, "is_email_verified": verified,
	}, nil)
}

// notices are the two-step notices sent to an address so far.
func (m *mailbox) notices(to string) []mail.Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	var found []mail.Message
	for _, msg := range m.messages {
		if strings.EqualFold(msg.To, to) && msg.Subject == noticeSubject {
			found = append(found, msg)
		}
	}
	return found
}

// waitNotices waits briefly for the nth two-step notice to an address.
func (m *mailbox) waitNotices(t *testing.T, to string, n int) []mail.Message {
	t.Helper()

	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if found := m.notices(to); len(found) >= n {
			return found
		}
	}

	t.Fatalf("%d two-step notices to %s, want %d", len(m.notices(to)), to, n)
	return nil
}

// persistent reports whether the response kept the user's session cookie past
// the browser's closing, and whether it set one at all.
func persistent(header http.Header) (kept, set bool) {
	for _, cookie := range (&http.Response{Header: header}).Cookies() {
		if cookie.Name == session.UserCookie && cookie.Value != "" {
			return !cookie.Expires.IsZero() || cookie.MaxAge > 0, true
		}
	}
	return false, false
}

// A person without a second factor, signing in to an application that
// requires one, sets one up on the second-factor page: named by the waiting
// sign-in's cookie alone, never by an application's token, and finished once.
// The factor answers the sign-in, and the account is told by email.
func TestLiveARequiredSecondFactorIsSetUpWhileSigningIn(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "social", "totp")
	vault := f.requiringApp(t)

	// The shop's access token, which setup has to refuse.
	verifier, challenge := pkce(t)
	shop := f.exchange(f.signIn(f.s.browser(), challenge).Query().Get("code"), verifier)

	b := f.s.browser()
	verifier, challenge = pkce(t)
	var waiting struct {
		SecondFactor bool `json:"second_factor"`
	}
	b.signInAs(adaEmail, adaPassword, page(t, b.visit(vault.authorizeURL(challenge, nil)), "/login"), false, &waiting)
	if !waiting.SecondFactor {
		t.Fatal("the sign-in did not wait for a factor to be set up")
	}
	if status := b.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Error("a session was opened before a factor was set up")
	}

	var offer enrolOffer
	if status := b.account(http.MethodGet, "/login/second-factor", nil, &offer); status != http.StatusOK ||
		!offer.SecondFactor.Enrol || offer.SecondFactor.StepUp || !slices.Equal(offer.SecondFactor.Methods, []string{"totp", "passkey"}) {
		t.Fatalf("the second-factor page = %d %+v, want setup of an authenticator or a passkey", status, offer.SecondFactor)
	}

	// An application's token never sets a factor up, even beside the cookie.
	var problem problemBody
	bearer := http.Header{"Authorization": {"Bearer " + shop.str("access_token")}}
	for _, path := range []string{"/login/second-factor/authenticator", "/login/second-factor/passkeys"} {
		if status, _ := b.accountWith(http.MethodPost, path, nil, bearer, fresh(&problem)); status != http.StatusForbidden || problem.Code != "browser_session_required" {
			t.Errorf("POST %s with an access token = %d %s, want browser_session_required", path, status, problem.Code)
		}
	}
	if started := f.s.count(t, "user_factors"); started != 0 {
		t.Errorf("%d factors were started with an access token, want none", started)
	}

	// Without the waiting sign-in's cookie there is nothing to set up.
	if status := f.s.browser().account(http.MethodPost, "/login/second-factor/authenticator", nil, fresh(&problem)); status != http.StatusGone || problem.Code != "second_factor_expired" {
		t.Errorf("setup without the cookie = %d %s, want second_factor_expired", status, problem.Code)
	}

	// Nothing to answer yet: the account has no factor a code could be from.
	if _, code := answerSecondFactor(b, map[string]any{"code": "123456"}); code != "second_factor_invalid" {
		t.Errorf("a code before any factor = %s, want second_factor_invalid", code)
	}

	var setup struct {
		Authenticator struct {
			Secret string `json:"secret"`
		} `json:"authenticator"`
	}
	if status := b.account(http.MethodPost, "/login/second-factor/authenticator", nil, &setup); status != http.StatusOK || setup.Authenticator.Secret == "" {
		t.Fatalf("starting an authenticator while signing in = %d", status)
	}
	if status := b.account(http.MethodPost, "/login/second-factor/authenticator/confirm", map[string]any{"code": "000000"}, fresh(&problem)); problem.Code != "second_factor_invalid" {
		t.Errorf("a wrong code from the new authenticator = %d %s, want second_factor_invalid", status, problem.Code)
	}

	waitingCookie := b.cookieOf(session.SecondFactorCookie, "/")
	code := map[string]any{"code": codeAt(t, setup.Authenticator.Secret, totp.Step(time.Now()))}
	var done enrolled
	if status := b.account(http.MethodPost, "/login/second-factor/authenticator/confirm", code, &done); status != http.StatusOK {
		t.Fatalf("confirming the authenticator while signing in = %d %s", status, done.Code)
	}
	if len(done.RecoveryCodes) != 10 {
		t.Errorf("got %d recovery codes with the first factor, want 10", len(done.RecoveryCodes))
	}
	back, _ := url.Parse(done.RedirectTo)
	tokens := vault.exchange(codeBack(t, back, vaultRedirect), verifier)
	if got, want := amrOf(t, tokens), []string{"pwd", "otp", "mfa"}; !slices.Equal(got, want) {
		t.Errorf("a password and a factor set up while signing in: amr = %v, want %v", got, want)
	}
	if status := b.account(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Error("the browser is not signed in after setting its factor up")
	}

	// The waiting sign-in is spent: its cookie, kept and sent again, sets
	// nothing more up.
	replay := f.s.browser()
	replay.setCookie(session.SecondFactorCookie, "/", waitingCookie)
	if status := replay.account(http.MethodPost, "/login/second-factor/authenticator", nil, fresh(&problem)); status != http.StatusGone || problem.Code != "second_factor_expired" {
		t.Errorf("the spent sign-in's cookie = %d %s, want second_factor_expired", status, problem.Code)
	}

	var twoStep struct {
		TwoStep struct {
			Enabled       bool `json:"enabled"`
			Authenticator *struct {
				ID string `json:"id"`
			} `json:"authenticator"`
		} `json:"two_step"`
	}
	b.account(http.MethodGet, "/two-step", nil, &twoStep)
	if !twoStep.TwoStep.Enabled || twoStep.TwoStep.Authenticator == nil {
		t.Errorf("after setup while signing in, two-step = %+v, want the authenticator on", twoStep.TwoStep)
	}

	notice := f.s.mail.waitNotices(t, adaEmail, 1)[0]
	if !strings.Contains(notice.Body, "an authenticator app") || !strings.Contains(notice.Body, adaEmail) {
		t.Errorf("the notice says %q, want the address and an authenticator app", notice.Body)
	}

	// The next sign-in asks for that factor, and offers nothing to set up.
	again := f.s.browser()
	_, challenge = pkce(t)
	again.signInAs(adaEmail, adaPassword, page(t, again.visit(vault.authorizeURL(challenge, nil)), "/login"), false, nil)
	again.account(http.MethodGet, "/login/second-factor", nil, &offer)
	if offer.SecondFactor.Enrol {
		t.Error("a sign-in by an account with a factor offered to set another up")
	}
	// Nor a passkey beside it: one added here would answer the very sign-in
	// that was to ask for the factor the account has.
	for _, path := range []string{"/login/second-factor/authenticator", "/login/second-factor/passkeys"} {
		if status := again.account(http.MethodPost, path, nil, fresh(&problem)); problem.Code != "two_step_not_offered" {
			t.Errorf("POST %s by an account that has a factor = %d %s, want two_step_not_offered", path, status, problem.Code)
		}
	}
}

// A passkey set up while signing in answers the sign-in as well.
func TestLiveAPasskeyCanBeSetUpWhileSigningIn(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)

	b := f.s.browser()
	verifier, challenge := pkce(t)
	b.signInAs(adaEmail, adaPassword, page(t, b.visit(vault.authorizeURL(challenge, nil)), "/login"), false, nil)

	key := newVirtualKey(t)
	var begun struct {
		Passkey map[string]any `json:"passkey"`
	}
	if status := b.account(http.MethodPost, "/login/second-factor/passkeys", nil, &begun); status != http.StatusOK {
		t.Fatalf("starting a passkey while signing in = %d", status)
	}
	var done enrolled
	if status := b.account(http.MethodPost, "/login/second-factor/passkeys/finish", map[string]any{
		"label": "Laptop", "response": key.create(t, begun.Passkey),
	}, &done); status != http.StatusOK || len(done.RecoveryCodes) != 10 {
		t.Fatalf("finishing a passkey while signing in = %d %s, %d recovery codes", status, done.Code, len(done.RecoveryCodes))
	}
	back, _ := url.Parse(done.RedirectTo)
	tokens := vault.exchange(codeBack(t, back, vaultRedirect), verifier)
	if got, want := amrOf(t, tokens), []string{"pwd", "hwk", "mfa"}; !slices.Equal(got, want) {
		t.Errorf("a password and a passkey set up while signing in: amr = %v, want %v", got, want)
	}

	notice := f.s.mail.waitNotices(t, adaEmail, 1)[0]
	if !strings.Contains(notice.Body, "a passkey") {
		t.Errorf("the notice says %q, want a passkey", notice.Body)
	}
}

// A signed-in person without a factor sets one up in the session they have,
// for an application that requires one: only from the browser holding that
// session. The session keeps its id under a new cookie, kept as long as the
// old one was.
func TestLiveASignedInPersonSetsUpAFactorInTheirSession(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)

	for _, remember := range []bool{true, false} {
		email := "grace@example.com"
		if !remember {
			email = "hopper@example.com"
		}
		f.makeUser(email, adaPassword, true)

		owner := f.s.browser()
		verifier, challenge := pkce(t)
		var first struct {
			RedirectTo string `json:"redirect_to"`
		}
		owner.signInAs(email, adaPassword, page(t, owner.visit(f.authorizeURL(challenge, nil)), "/login"), remember, &first)
		back, _ := url.Parse(first.RedirectTo)
		shop := f.exchange(codeBack(t, back, shopRedirect), verifier)
		_, shopClaims := f.verifyJWT(shop.str("id_token"))

		other := f.s.browser()
		other.signInAs(email, adaPassword, "", false, nil)

		verifier, challenge = pkce(t)
		page(t, owner.visit(vault.authorizeURL(challenge, nil)), "/second-factor")
		var offer enrolOffer
		owner.account(http.MethodGet, "/login/second-factor", nil, &offer)
		if !offer.SecondFactor.Enrol || !offer.SecondFactor.StepUp {
			t.Fatalf("remember %t: the second-factor page = %+v, want setup in this session", remember, offer.SecondFactor)
		}

		// The waiting step-up's cookie, copied to a browser with no session
		// or with another of the person's sessions, sets nothing up.
		stepUp := owner.cookieOf(session.SecondFactorCookie, "/")
		var problem problemBody
		for name, b := range map[string]*browser{"no session": f.s.browser(), "another session": other} {
			b.setCookie(session.SecondFactorCookie, "/", stepUp)
			if status := b.account(http.MethodPost, "/login/second-factor/authenticator", nil, fresh(&problem)); status != http.StatusGone || problem.Code != "second_factor_expired" {
				t.Errorf("remember %t: setup from a browser with %s = %d %s, want second_factor_expired", remember, name, status, problem.Code)
			}
		}
		if started := f.s.count(t, "user_factors"); started != 0 && remember {
			t.Errorf("%d factors were started from the wrong browsers, want none", started)
		}

		before := owner.cookieOf(session.UserCookie, "/")
		var setup struct {
			Authenticator struct {
				Secret string `json:"secret"`
			} `json:"authenticator"`
		}
		owner.account(http.MethodPost, "/login/second-factor/authenticator", nil, &setup)
		var done enrolled
		status, header := owner.accountWith(http.MethodPost, "/login/second-factor/authenticator/confirm",
			map[string]any{"code": codeAt(t, setup.Authenticator.Secret, totp.Step(time.Now()))}, nil, &done)
		if status != http.StatusOK {
			t.Fatalf("remember %t: setting up in the session = %d %s", remember, status, done.Code)
		}
		if kept, set := persistent(header); !set || kept != remember {
			t.Errorf("remember %t: the stepped-up cookie set %t, kept past the browser %t", remember, set, kept)
		}

		back, _ = url.Parse(done.RedirectTo)
		tokens := vault.exchange(codeBack(t, back, vaultRedirect), verifier)
		_, claims := f.verifyJWT(tokens.str("id_token"))
		if claims["sid"] != shopClaims["sid"] {
			t.Errorf("remember %t: sid = %v, want the session's own %v", remember, claims["sid"], shopClaims["sid"])
		}
		if got, want := amrOf(t, tokens), []string{"pwd", "otp", "mfa"}; !slices.Equal(got, want) {
			t.Errorf("remember %t: amr = %v, want %v", remember, got, want)
		}
		if after := owner.cookieOf(session.UserCookie, "/"); after == before || after == "" {
			t.Errorf("remember %t: the session's cookie was not replaced", remember)
		}
		f.s.mail.waitNotices(t, email, 1)
	}
}

// A step-up from a session made with "stay signed in" keeps its new cookie
// as long; one made without, for the browser's lifetime.
func TestLiveAStepUpKeepsStaySignedIn(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)

	sessions := map[bool]*browser{true: f.s.browser(), false: f.s.browser()}
	for remember, b := range sessions {
		if status := b.signInAs(adaEmail, adaPassword, "", remember, nil); status != http.StatusOK {
			t.Fatalf("signing in = %d", status)
		}
	}
	_, codes := addAuthenticator(t, sessions[true])

	for i, remember := range []bool{true, false} {
		b := sessions[remember]
		_, challenge := pkce(t)
		page(t, b.visit(vault.authorizeURL(challenge, nil)), "/second-factor")

		var out struct {
			RedirectTo string `json:"redirect_to"`
		}
		status, header := b.accountWith(http.MethodPost, "/login/second-factor", map[string]any{"code": codes[i]}, nil, &out)
		if status != http.StatusOK {
			t.Fatalf("remember %t: the step-up = %d", remember, status)
		}
		if kept, set := persistent(header); !set || kept != remember {
			t.Errorf("remember %t: the stepped-up cookie set %t, kept past the browser %t", remember, set, kept)
		}
	}
}

// Setting a factor up while signing in needs a proved address: without one
// the verification email goes first, and nothing waits to be set up.
func TestLiveSettingUpAFactorNeedsAProvedAddress(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	allowUnverifiedAddresses(t, f.super)
	vault := f.requiringApp(t)
	f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+vault.flowID, map[string]any{"require_verified_email": false}, nil)

	const email = "unproved@example.com"
	f.makeUser(email, adaPassword, false)

	b := f.s.browser()
	_, challenge := pkce(t)
	request := page(t, b.visit(vault.authorizeURL(challenge, nil)), "/login")
	var problem problemBody
	if status := b.signInAs(email, adaPassword, request, false, fresh(&problem)); status != http.StatusForbidden || problem.Code != "email_not_verified" {
		t.Fatalf("signing in at an unproved address = %d %s, want email_not_verified", status, problem.Code)
	}
	link := linkIn(t, f.s.mail.waitFor(t, email, "/verify-email").Body, "/verify-email")
	if link.Query().Get("request") != request {
		t.Errorf("the verification link continues %q, want the sign-in %q", link.Query().Get("request"), request)
	}
	if held := b.cookieOf(session.SecondFactorCookie, "/"); held != "" {
		t.Error("a sign-in at an unproved address was held to set a factor up")
	}
	if status := b.account(http.MethodPost, "/login/second-factor/authenticator", nil, fresh(&problem)); problem.Code != "second_factor_expired" {
		t.Errorf("setup at an unproved address = %d %s, want second_factor_expired", status, problem.Code)
	}

	// Signed in already, through a flow that lets the address in: the
	// requiring application sends the browser to sign in again, which sends
	// the email, rather than to set a factor up.
	signedIn := f.s.browser()
	if status := signedIn.signInAs(email, adaPassword, "", false, nil); status != http.StatusOK {
		t.Fatalf("signing in to the account page = %d", status)
	}
	page(t, signedIn.visit(vault.authorizeURL(challenge, nil)), "/login")
	if started := f.s.count(t, "user_factors"); started != 0 {
		t.Errorf("%d factors were started for an unproved address, want none", started)
	}
}

// Every factor confirmed on the account page tells the account, too.
func TestLiveEveryNewFactorIsMailedToTheAccount(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")

	owner := f.s.browser()
	if status := owner.signInAs(adaEmail, adaPassword, "", false, nil); status != http.StatusOK {
		t.Fatalf("signing in = %d", status)
	}

	addAuthenticator(t, owner)
	first := f.s.mail.waitNotices(t, adaEmail, 1)
	if !strings.Contains(first[0].Body, "an authenticator app") {
		t.Errorf("the first notice says %q, want an authenticator app", first[0].Body)
	}

	// A second factor while one is on has to be confirmed with the password
	// and its own ceremony, and is told of as well.
	addPasskey(t, owner)
	both := f.s.mail.waitNotices(t, adaEmail, 2)
	if !strings.Contains(both[1].Body, "a passkey") {
		t.Errorf("the second notice says %q, want a passkey", both[1].Body)
	}
}

// Setting up while signing in is only for a flow that requires a factor: a
// flow that only offers one leaves it to the account page.
func TestLiveSetupWhileSigningInOnlyWhereAFactorIsRequired(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")

	b := f.s.browser()
	_, challenge := pkce(t)
	if f.signIn(b, challenge).Query().Get("code") == "" {
		t.Fatal("a flow that only offers a factor held a sign-in without one")
	}
	var problem problemBody
	if status := b.account(http.MethodPost, "/login/second-factor/authenticator", nil, fresh(&problem)); problem.Code != "second_factor_expired" {
		t.Errorf("setup with no waiting sign-in = %d %s, want second_factor_expired", status, problem.Code)
	}

	// A sign-in held for an existing factor, through the optional flow, has
	// nothing to set up.
	addAuthenticator(t, b)
	held := f.s.browser()
	held.signInAs(adaEmail, adaPassword, page(t, held.visit(f.authorizeURL(challenge, nil)), "/login"), false, nil)
	if status := held.account(http.MethodPost, "/login/second-factor/authenticator", nil, fresh(&problem)); problem.Code != "two_step_not_offered" {
		t.Errorf("setup in a sign-in through a flow that only offers a factor = %d %s, want two_step_not_offered", status, problem.Code)
	}

	// Still not once the factor it waits for is removed meanwhile: that flow
	// requires none, so its sign-in sets none up.
	var twoStep struct {
		TwoStep struct {
			Authenticator struct {
				ID string `json:"id"`
			} `json:"authenticator"`
		} `json:"two_step"`
	}
	b.account(http.MethodGet, "/two-step", nil, &twoStep)
	if status := b.account(http.MethodDelete, "/two-step/factors/"+twoStep.TwoStep.Authenticator.ID, map[string]any{"current_password": adaPassword}, nil); status != http.StatusNoContent {
		t.Fatalf("removing the authenticator = %d", status)
	}
	if status := held.account(http.MethodPost, "/login/second-factor/passkeys", nil, fresh(&problem)); problem.Code != "two_step_not_offered" {
		t.Errorf("setup in a sign-in through a flow that requires none = %d %s, want two_step_not_offered", status, problem.Code)
	}
}

// A sign-in waiting for a factor to be set up sets none up once its address
// is no longer proved (the email goes out again), nor while it holds a
// single sign-on identity to link, which waits for a factor the account had.
func TestLiveSetupWhileSigningInRefusesWhatItCannotTrust(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)

	waiting := func() *browser {
		b := f.s.browser()
		_, challenge := pkce(t)
		var held struct {
			SecondFactor bool `json:"second_factor"`
		}
		b.signInAs(adaEmail, adaPassword, page(t, b.visit(vault.authorizeURL(challenge, nil)), "/login"), false, &held)
		if !held.SecondFactor {
			t.Fatal("the sign-in did not wait for a factor to be set up")
		}
		return b
	}
	var problem problemBody

	unproved := waiting()
	if err := f.s.db.Exec("UPDATE users SET is_email_verified = false WHERE email = ?", adaEmail).Error; err != nil {
		t.Fatal(err)
	}
	if status := unproved.account(http.MethodPost, "/login/second-factor/authenticator", nil, fresh(&problem)); problem.Code != "email_not_verified" {
		t.Errorf("setup at an address no longer proved = %d %s, want email_not_verified", status, problem.Code)
	}
	f.s.mail.waitFor(t, adaEmail, "/verify-email")
	if err := f.s.db.Exec("UPDATE users SET is_email_verified = true WHERE email = ?", adaEmail).Error; err != nil {
		t.Fatal(err)
	}

	linking := waiting()
	idp := newFakeOIDC(t)
	var created struct {
		Connection struct {
			ID string `json:"id"`
		} `json:"connection"`
	}
	f.super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Example", "issuer": idp.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret,
		"domains": []string{"example.com"},
	}, &created)
	if err := f.s.db.Exec("UPDATE pending_sign_ins SET link_connection_id = ?, link_subject = 'ada' WHERE used_at IS NULL",
		created.Connection.ID).Error; err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/login/second-factor/authenticator", "/login/second-factor/passkeys"} {
		if status := linking.account(http.MethodPost, path, nil, fresh(&problem)); problem.Code != "two_step_not_offered" {
			t.Errorf("POST %s in a sign-in holding a link = %d %s, want two_step_not_offered", path, status, problem.Code)
		}
	}
	if started := f.s.count(t, "user_factors"); started != 0 {
		t.Errorf("%d factors were started, want none", started)
	}
}

// The account page offers two-step verification wherever a flow in use has
// its step: the default flow, or another while it is on.
func TestLiveTwoStepIsOfferedWhereAFlowInUseHasIt(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password")

	owner := f.s.browser()
	if status := owner.signInAs(adaEmail, adaPassword, "", false, nil); status != http.StatusOK {
		t.Fatalf("signing in = %d", status)
	}
	offered := func() bool {
		var out struct {
			TwoStep struct {
				Offered bool `json:"offered"`
			} `json:"two_step"`
		}
		owner.account(http.MethodGet, "/two-step", nil, &out)
		return out.TwoStep.Offered
	}

	if offered() {
		t.Error("two-step is offered with no flow that has its step")
	}

	vault := f.requiringApp(t)
	if !offered() {
		t.Error("two-step is not offered while an application's flow requires it")
	}

	f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+vault.flowID, map[string]any{"is_enabled": false}, nil)
	if offered() {
		t.Error("two-step is offered by a flow that is off")
	}
	var problem problemBody
	if status := owner.account(http.MethodPost, "/two-step/authenticator", map[string]any{"current_password": adaPassword}, fresh(&problem)); problem.Code != "two_step_not_offered" {
		t.Errorf("adding an authenticator with no flow offering it = %d %s, want two_step_not_offered", status, problem.Code)
	}
}

// count is how many rows a table holds.
func (s *liveServer) count(t *testing.T, table string) int64 {
	t.Helper()
	var n int64
	if err := s.db.Table(table).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

// fresh empties a problem before an answer is decoded into it, so a call
// that succeeds cannot leave the last refusal's code in place.
func fresh(problem *problemBody) *problemBody {
	*problem = problemBody{}
	return problem
}
