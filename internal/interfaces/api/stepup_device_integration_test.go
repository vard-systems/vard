package api

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"vard/internal/domain/model"
	"vard/internal/domain/totp"
	"vard/internal/interfaces/api/kit/session"
)

// A device, a refresh and leaving a second factor for later, each held to the
// flow of the application they are for, and two setups for one account at
// once.

// requiringDevice registers a device client on the vault's flow, which
// requires a second factor, and starts a device; it returns the codes and a
// poll for the device's tokens.
func (f *oauthFixture) requiringDevice(t *testing.T, vault requiringApp) (userCode string, poll func() tokenResult) {
	t.Helper()

	tvID, _, _ := f.register(map[string]any{
		"name": "Vault TV", "type": "native",
		"grant_types": []string{"urn:ietf:params:oauth:grant-type:device_code", "refresh_token"},
		"scopes":      []string{"openid", "email", "offline_access"},
	})
	flow, err := uuid.Parse(vault.flowID)
	if err != nil {
		t.Fatal(err)
	}
	f.s.changeApplication(tvID, func(app *model.Application) { app.LoginFlowID = &flow })

	started := f.token("/oauth2/device_authorization", tvID, "", url.Values{"scope": {"openid email offline_access"}})
	if started.status != http.StatusOK {
		t.Fatalf("starting a device = %d %v", started.status, started.body)
	}
	deviceCode := started.str("device_code")

	return started.str("user_code"), func() tokenResult {
		return f.token("/oauth2/token", tvID, "", url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {deviceCode},
		})
	}
}

// startAuthenticator begins an authenticator in b's waiting sign-in and
// returns its key.
func startAuthenticator(t *testing.T, b *browser) string {
	t.Helper()

	var setup struct {
		Authenticator struct {
			Secret string `json:"secret"`
		} `json:"authenticator"`
	}
	if status := b.account(http.MethodPost, "/login/second-factor/authenticator", nil, &setup); status != http.StatusOK {
		t.Fatalf("starting an authenticator while signing in = %d", status)
	}
	return setup.Authenticator.Secret
}

// Allowing a device for an application whose flow requires a second factor
// asks a session without one to pass one (or set one up) first: the device
// page starts a step-up, the second-factor page finishes it, and the device
// is allowed after. Only from the person's own browser.
func TestLiveADeviceIsAllowedOnlyAsItsFlowAsks(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)
	userCode, poll := f.requiringDevice(t, vault)

	b := f.s.browser()
	if status := b.signInAs(adaEmail, adaPassword, "", false, nil); status != http.StatusOK {
		t.Fatalf("signing in = %d", status)
	}

	var problem problemBody
	if status := b.account(http.MethodPost, "/device", map[string]any{"user_code": userCode, "yes": true}, fresh(&problem)); status != http.StatusForbidden || problem.Code != "step_up_required" {
		t.Fatalf("allowing the device with a password alone = %d %s, want step_up_required", status, problem.Code)
	}
	if got := poll().str("error"); got != "authorization_pending" {
		t.Errorf("after the refusal the device is told %s, want authorization_pending", got)
	}

	var started struct {
		SecondFactor bool `json:"second_factor"`
	}
	if status := b.account(http.MethodPost, "/step-up", map[string]any{"user_code": userCode}, &started); status != http.StatusOK || !started.SecondFactor {
		t.Fatalf("starting the step-up = %d %+v", status, started)
	}
	var offer enrolOffer
	b.account(http.MethodGet, "/login/second-factor", nil, &offer)
	if !offer.SecondFactor.StepUp || !offer.SecondFactor.Enrol {
		t.Fatalf("the second-factor page = %+v, want a factor set up in this session", offer.SecondFactor)
	}

	// The waiting step-up is held to the device's flow, which requires a
	// factor, not the default one, which only offers it.
	secret := startAuthenticator(t, b)
	var done enrolled
	if status := b.account(http.MethodPost, "/login/second-factor/authenticator/confirm",
		map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}, &done); status != http.StatusOK || done.RedirectTo != "" {
		t.Fatalf("setting the factor up = %d %+v, want no redirect", status, done)
	}

	if status := b.account(http.MethodPost, "/device", map[string]any{"user_code": userCode, "yes": true}, nil); status != http.StatusNoContent {
		t.Fatalf("allowing the device after the step-up = %d", status)
	}
	tokens := poll()
	if got, want := amrOf(t, tokens), []string{"pwd", "otp", "mfa"}; !slices.Equal(got, want) {
		t.Errorf("the device's amr = %v, want %v", got, want)
	}
}

// A refresh family that never passed a second factor ends once its
// application's flow comes to require one, a retry within the grace too; a
// family that passed one goes on.
func TestLiveARefreshEndsWhenItsFlowComesToRequireASecondFactor(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")

	b := f.s.browser()
	verifier, challenge := pkce(t)
	first := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier)
	replaced := first.str("refresh_token")
	second := f.refresh(replaced)
	if second.status != http.StatusOK {
		t.Fatalf("refreshing before the requirement = %d %v", second.status, second.body)
	}

	// The family that will pass a factor: ada adds one and signs in with it.
	secret, _ := addAuthenticator(t, b)
	strong := f.s.browser()
	verifier, challenge = pkce(t)
	var held struct {
		SecondFactor bool `json:"second_factor"`
	}
	strong.signInAs(adaEmail, adaPassword, page(t, strong.visit(f.authorizeURL(challenge, nil)), "/login"), false, &held)
	status, to := answerSecondFactor(strong, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))})
	if status != http.StatusOK {
		t.Fatalf("the second factor = %d %s", status, to)
	}
	back, _ := url.Parse(to)
	passed := f.exchange(codeBack(t, back, shopRedirect), verifier)

	f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+defaultFlow(t, f.super), map[string]any{"require_second_factor": true}, nil)

	for name, token := range map[string]string{
		"the family's live token":            second.str("refresh_token"),
		"a retry of the token just replaced": replaced,
	} {
		if got := f.refresh(token); got.status != http.StatusBadRequest || got.str("error") != "invalid_grant" {
			t.Errorf("%s once a factor is required = %d %v, want invalid_grant", name, got.status, got.body)
		}
	}

	if got := f.refresh(passed.str("refresh_token")); got.status != http.StatusOK {
		t.Errorf("a family that passed a factor = %d %v, want 200", got.status, got.body)
	}
}

// "Not now" on a step-up answers the application with access_denied and the
// state it sent, once: leaving again, answering the factor or setting one up
// with the same cookie is refused afterwards, and only the browser holding
// the session may leave. Without an authorization request (a device) it
// goes to the account page.
func TestLiveNotNowAnswersTheApplication(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)

	owner := f.s.browser()
	if status := owner.signInAs(adaEmail, adaPassword, "", false, nil); status != http.StatusOK {
		t.Fatalf("signing in = %d", status)
	}
	other := f.s.browser()
	other.signInAs(adaEmail, adaPassword, "", false, nil)
	secret, recoveryCodes := addAuthenticator(t, owner)

	_, challenge := pkce(t)
	request := page(t, owner.visit(vault.authorizeURL(challenge, nil)), "/second-factor")
	stepUp := owner.cookieOf(session.SecondFactorCookie, "/")

	var problem problemBody
	for name, b := range map[string]*browser{"no session": f.s.browser(), "another session": other} {
		b.setCookie(session.SecondFactorCookie, "/", stepUp)
		if status := b.account(http.MethodPost, "/login/second-factor/leave", nil, fresh(&problem)); status != http.StatusGone || problem.Code != "second_factor_expired" {
			t.Errorf("leaving from a browser with %s = %d %s, want second_factor_expired", name, status, problem.Code)
		}
	}
	bearer := http.Header{"Authorization": {"Bearer token"}}
	if status, _ := owner.accountWith(http.MethodPost, "/login/second-factor/leave", nil, bearer, fresh(&problem)); status != http.StatusForbidden || problem.Code != "browser_session_required" {
		t.Errorf("leaving with a bearer token = %d %s, want browser_session_required", status, problem.Code)
	}

	var left enrolled
	if status := owner.account(http.MethodPost, "/login/second-factor/leave", nil, &left); status != http.StatusOK {
		t.Fatalf("leaving = %d %s", status, left.Code)
	}
	back, _ := url.Parse(left.RedirectTo)
	if !strings.HasPrefix(back.String(), vaultRedirect+"?") || back.Query().Get("error") != "access_denied" ||
		back.Query().Get("state") != "vault-state" || back.Query().Get("iss") != f.s.root {
		t.Errorf("leaving sent the browser to %s, want the vault with access_denied, its state and iss", back)
	}
	if owner.cookieOf(session.SecondFactorCookie, "/") != "" {
		t.Error("leaving kept the waiting sign-in's cookie")
	}

	// The same cookie, kept and sent again, does nothing more.
	owner.setCookie(session.SecondFactorCookie, "/", stepUp)
	if status := owner.account(http.MethodPost, "/login/second-factor/leave", nil, fresh(&problem)); status != http.StatusGone || problem.Code != "second_factor_expired" {
		t.Errorf("leaving twice = %d %s, want second_factor_expired", status, problem.Code)
	}
	if status, code := answerSecondFactor(owner, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}); status != http.StatusGone || code != "second_factor_expired" {
		t.Errorf("answering after leaving = %d %s, want second_factor_expired", status, code)
	}

	// Nor does the request: the application was told no, so signing in to it
	// afresh gives it no code after all.
	afresh := f.s.browser()
	afresh.signInAs(adaEmail, adaPassword, request, false, nil)
	if status, code := answerSecondFactor(afresh, map[string]any{"code": recoveryCodes[0]}); code != "request_expired" {
		t.Errorf("signing in to the request that was left = %d %s, want request_expired", status, code)
	}

	// A factor answered first leaves nothing to leave, and the application
	// is not told access_denied after its code.
	_, challenge = pkce(t)
	page(t, owner.visit(vault.authorizeURL(challenge, nil)), "/second-factor")
	answered := owner.cookieOf(session.SecondFactorCookie, "/")
	if status, to := answerSecondFactor(owner, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now())+1)}); status != http.StatusOK {
		t.Fatalf("answering the step-up = %d %s", status, to)
	}
	owner.setCookie(session.SecondFactorCookie, "/", answered)
	if status := owner.account(http.MethodPost, "/login/second-factor/leave", nil, fresh(&problem)); status != http.StatusGone || problem.Code != "second_factor_expired" {
		t.Errorf("leaving after answering = %d %s, want second_factor_expired", status, problem.Code)
	}

	// A step-up for a device has no application waiting at a redirect.
	grace := f.s.browser()
	f.makeUser("grace@example.com", adaPassword, true)
	grace.signInAs("grace@example.com", adaPassword, "", false, nil)
	userCode, _ := f.requiringDevice(t, vault)
	grace.account(http.MethodPost, "/step-up", map[string]any{"user_code": userCode}, nil)
	if status := grace.account(http.MethodPost, "/login/second-factor/leave", nil, &left); status != http.StatusOK || left.RedirectTo != testAccountURL+"/" {
		t.Errorf("leaving a device's step-up = %d %s, want the account page", status, left.RedirectTo)
	}
}

// Leaving a factor's setup for later spends it: setting it up afterwards
// with the same cookie is refused, and nothing is kept.
func TestLiveNotNowEndsASetup(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)

	b := f.s.browser()
	b.signInAs(adaEmail, adaPassword, "", false, nil)
	_, challenge := pkce(t)
	page(t, b.visit(vault.authorizeURL(challenge, nil)), "/second-factor")
	waiting := b.cookieOf(session.SecondFactorCookie, "/")
	secret := startAuthenticator(t, b)

	var left enrolled
	if status := b.account(http.MethodPost, "/login/second-factor/leave", nil, &left); status != http.StatusOK ||
		!strings.Contains(left.RedirectTo, "error=access_denied") {
		t.Fatalf("leaving the setup = %d %s", status, left.RedirectTo)
	}

	b.setCookie(session.SecondFactorCookie, "/", waiting)
	var problem problemBody
	if status := b.account(http.MethodPost, "/login/second-factor/authenticator/confirm",
		map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}, fresh(&problem)); status != http.StatusGone || problem.Code != "second_factor_expired" {
		t.Errorf("setting up after leaving = %d %s, want second_factor_expired", status, problem.Code)
	}
	if confirmed := f.s.count(t, "user_factors WHERE confirmed_at IS NOT NULL"); confirmed != 0 {
		t.Errorf("%d factors were kept after leaving, want none", confirmed)
	}
}

// Two sign-ins setting up an account's first factor at once keep one: the
// other is refused, never a second factor nobody was asked for.
func TestLiveTwoSetupsAtOnceKeepOneFactor(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)

	for round := range 5 {
		email := "round" + string(rune('a'+round)) + "@example.com"
		f.makeUser(email, adaPassword, true)

		waiting := func() *browser {
			b := f.s.browser()
			_, challenge := pkce(t)
			var held struct {
				SecondFactor bool `json:"second_factor"`
			}
			b.signInAs(email, adaPassword, page(t, b.visit(vault.authorizeURL(challenge, nil)), "/login"), false, &held)
			if !held.SecondFactor {
				t.Fatal("the sign-in did not wait for a factor to be set up")
			}
			return b
		}
		app, phone := waiting(), waiting()

		// One sets up an authenticator, the other a passkey: two
		// unconfirmed factors, each with its own ceremony.
		secret := startAuthenticator(t, app)
		key := newVirtualKey(t)
		var begun struct {
			Passkey map[string]any `json:"passkey"`
		}
		if status := phone.account(http.MethodPost, "/login/second-factor/passkeys", nil, &begun); status != http.StatusOK {
			t.Fatalf("starting a passkey = %d", status)
		}
		created := key.create(t, begun.Passkey)

		statuses := make([]int, 2)
		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			statuses[0] = app.account(http.MethodPost, "/login/second-factor/authenticator/confirm",
				map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}, nil)
		}()
		go func() {
			defer wg.Done()
			<-start
			statuses[1] = phone.account(http.MethodPost, "/login/second-factor/passkeys/finish",
				map[string]any{"label": "Phone", "response": created}, nil)
		}()
		close(start)
		wg.Wait()

		var confirmed int64
		if err := f.s.db.Raw(`SELECT count(*) FROM user_factors f JOIN users u ON u.id = f.user_id
			WHERE u.email = ? AND f.confirmed_at IS NOT NULL`, email).Scan(&confirmed).Error; err != nil {
			t.Fatal(err)
		}
		ok := 0
		for _, status := range statuses {
			if status == http.StatusOK {
				ok++
			}
		}
		if confirmed != 1 || ok != 1 {
			t.Errorf("round %d: two setups at once kept %d factors with answers %v, want one kept and one refused", round, confirmed, statuses)
		}
	}
}

// A device's step-up for an account without a factor sets one up only at a
// proved address: the verification email goes first.
func TestLiveADeviceStepUpNeedsAProvedAddress(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	allowUnverifiedAddresses(t, f.super)
	vault := f.requiringApp(t)
	f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+vault.flowID, map[string]any{"require_verified_email": false}, nil)
	userCode, _ := f.requiringDevice(t, vault)

	const email = "unproved@example.com"
	f.makeUser(email, adaPassword, false)
	b := f.s.browser()
	if status := b.signInAs(email, adaPassword, "", false, nil); status != http.StatusOK {
		t.Fatalf("signing in = %d", status)
	}

	var problem problemBody
	if status := b.account(http.MethodPost, "/step-up", map[string]any{"user_code": userCode}, fresh(&problem)); status != http.StatusForbidden || problem.Code != "email_not_verified" {
		t.Errorf("a device step-up at an unproved address = %d %s, want email_not_verified", status, problem.Code)
	}
	f.s.mail.waitFor(t, email, "/verify-email")
	if held := b.cookieOf(session.SecondFactorCookie, "/"); held != "" {
		t.Error("a step-up at an unproved address was held to set a factor up")
	}
}
