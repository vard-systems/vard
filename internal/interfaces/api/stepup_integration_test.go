package api

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"vard/internal/domain/model"
	"vard/internal/domain/totp"
	"vard/internal/infrastructure/cache"
	"vard/internal/interfaces/api/kit/session"
)

// An application whose login flow requires a second factor, beside the
// fixture's shop, whose default flow only offers one.

const vaultRedirect = "https://vault.example.com/callback"

type requiringApp struct {
	f        *oauthFixture
	clientID string
	secret   string
	flowID   string
}

func (f *oauthFixture) requiringApp(t *testing.T) requiringApp {
	t.Helper()

	clientID, secret, _ := f.register(map[string]any{
		"name": "Vault", "type": "web",
		"grant_types":   []string{"authorization_code", "refresh_token"},
		"redirect_uris": []string{vaultRedirect},
		"scopes":        []string{"openid", "profile", "email", "offline_access"},
	})

	var flow struct {
		Flow struct {
			ID                  string `json:"id"`
			RequireSecondFactor bool   `json:"require_second_factor"`
		} `json:"flow"`
	}
	f.super.must(http.StatusCreated, http.MethodPost, "/login-flows", map[string]any{
		"name": "Vault sign-in", "slug": "vault", "is_enabled": true,
		"steps":                 []string{"identifier", "password", "totp"},
		"require_second_factor": true,
	}, &flow)
	if !flow.Flow.RequireSecondFactor {
		t.Fatal("the flow was saved without requiring a second factor")
	}
	id, err := uuid.Parse(flow.Flow.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.s.changeApplication(clientID, func(app *model.Application) { app.LoginFlowID = &id })

	return requiringApp{f: f, clientID: clientID, secret: secret, flowID: flow.Flow.ID}
}

func (a requiringApp) authorizeURL(challenge string, extra url.Values) string {
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {a.clientID},
		"redirect_uri":          {vaultRedirect},
		"scope":                 {"openid profile email offline_access"},
		"state":                 {"vault-state"},
		"nonce":                 {"vault-nonce"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	for key, values := range extra {
		query[key] = values
	}

	return a.f.s.root + "/oauth2/authorize?" + query.Encode()
}

func (a requiringApp) exchange(code, verifier string) tokenResult {
	return a.f.token("/oauth2/token", a.clientID, a.secret, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {vaultRedirect},
		"code_verifier": {verifier},
	})
}

// cookieOf is the value of one of the browser's cookies at path, "" for none.
func (b *browser) cookieOf(name, path string) string {
	at, _ := url.Parse(b.s.root + path)
	for _, cookie := range b.http.Jar.Cookies(at) {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

// setCookie puts a cookie in the browser, as one copied from another would be.
func (b *browser) setCookie(name, path, value string) {
	at, _ := url.Parse(b.s.root + path)
	b.http.Jar.SetCookies(at, []*http.Cookie{{Name: name, Value: value, Path: path}})
}

// ageSession moves the start of the session the browser holds back by d,
// leaving when it last authenticated alone, past any copy in the cache.
func (f *oauthFixture) ageSession(t *testing.T, b *browser, d time.Duration) {
	t.Helper()

	hash := model.HashSecret(b.cookieOf(session.UserCookie, "/"))
	if err := f.s.db.Exec("UPDATE user_sessions SET created_at = created_at - make_interval(secs => ?) WHERE token_hash = ?", d.Seconds(), hash).Error; err != nil {
		t.Fatal(err)
	}
	if f.s.cache != nil {
		f.s.cache.SessionDB().DropSessions(context.Background(), cache.UserSession, hash)
	}
}

// codeBack is the code in a redirect to the application, failing the test
// when the browser went anywhere else.
func codeBack(t *testing.T, at *url.URL, redirect string) string {
	t.Helper()

	if !strings.HasPrefix(at.String(), redirect+"?") || at.Query().Get("code") == "" {
		t.Fatalf("the browser went to %s, want %s with a code", at, redirect)
	}
	return at.Query().Get("code")
}

// A person signed in with a password alone, who has an authenticator, is
// asked for it before an application that requires one: in the session they
// have, which keeps its id and gets a new cookie. A step-up's cookie alone,
// in a browser without that session, upgrades nothing.
func TestLiveAnApplicationCanRequireASecondFactor(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "social", "totp")
	vault := f.requiringApp(t)

	owner := f.s.browser()
	verifier, challenge := pkce(t)
	shop := f.exchange(f.signIn(owner, challenge).Query().Get("code"), verifier)
	_, shopClaims := f.verifyJWT(shop.str("id_token"))

	// Another session of the same person, made with a password too.
	other := f.s.browser()
	f.signIn(other, challenge)

	secret, _ := addAuthenticator(t, owner)

	// prompt=none shows nothing, so the client is told what was needed.
	silent := owner.visit(vault.authorizeURL(challenge, url.Values{"prompt": {"none"}}))
	if got := silent.Query().Get("error"); !strings.HasPrefix(silent.String(), vaultRedirect+"?") || got != "interaction_required" {
		t.Errorf("prompt=none before a step-up went to %s, want interaction_required", silent)
	}

	verifier, challenge = pkce(t)
	asked := owner.visit(vault.authorizeURL(challenge, nil))
	page(t, asked, "/second-factor")

	var offer struct {
		SecondFactor struct {
			StepUp  bool     `json:"step_up"`
			Methods []string `json:"methods"`
		} `json:"second_factor"`
	}
	if status := owner.account(http.MethodGet, "/login/second-factor", nil, &offer); status != http.StatusOK || !offer.SecondFactor.StepUp {
		t.Fatalf("the second-factor page = %d %+v, want a step-up", status, offer)
	}

	before := owner.cookieOf(session.UserCookie, "/")
	stepUpCookie := owner.cookieOf(session.SecondFactorCookie, "/")
	if stepUpCookie == "" {
		t.Fatal("authorize set no cookie naming the step-up")
	}
	code := map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}

	// The step-up's cookie, copied into a browser with no session or with
	// another of the same person's sessions, is refused, and the code it
	// carried is not spent.
	for name, b := range map[string]*browser{"no session": f.s.browser(), "another session": other} {
		b.setCookie(session.SecondFactorCookie, "/", stepUpCookie)
		if status, problem := answerSecondFactor(b, code); status != http.StatusGone || problem != "second_factor_expired" {
			t.Errorf("a step-up answered from a browser with %s = %d %s, want second_factor_expired", name, status, problem)
		}
	}

	stepped := time.Now().Truncate(time.Second)
	status, to := answerSecondFactor(owner, code)
	if status != http.StatusOK {
		t.Fatalf("the step-up = %d %s", status, to)
	}
	back, _ := url.Parse(to)
	tokens := vault.exchange(codeBack(t, back, vaultRedirect), verifier)
	if tokens.status != http.StatusOK {
		t.Fatalf("exchanging the code = %d %v", tokens.status, tokens.body)
	}

	if got, want := amrOf(t, tokens), []string{"pwd", "otp", "mfa"}; !slices.Equal(got, want) {
		t.Errorf("after the step-up, amr = %v, want %v", got, want)
	}
	_, claims := f.verifyJWT(tokens.str("id_token"))
	if claims["sid"] != shopClaims["sid"] || claims["sid"] == nil {
		t.Errorf("the step-up's sid = %v, want the session's own %v", claims["sid"], shopClaims["sid"])
	}
	if authTime, _ := claims["auth_time"].(float64); int64(authTime) < stepped.Unix() {
		t.Errorf("auth_time = %v, want the time of the step-up, %d or later", claims["auth_time"], stepped.Unix())
	}

	// The cookie seen before the second factor is worth nothing after it.
	after := owner.cookieOf(session.UserCookie, "/")
	if after == "" || after == before {
		t.Fatal("the step-up left the session's cookie as it was")
	}
	stale := f.s.browser()
	stale.setCookie(session.UserCookie, "/", before)
	if status := stale.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Error("the cookie from before the step-up still names the session")
	}

	// What was issued in the session before goes on working.
	if status := f.userinfoStatus(shop.str("access_token")); status != http.StatusOK {
		t.Errorf("the shop's access token after the step-up = %d, want 200", status)
	}

	// The session has passed a second factor now: no step-up again.
	_, challenge = pkce(t)
	codeBack(t, owner.visit(vault.authorizeURL(challenge, nil)), vaultRedirect)
}

// A step-up renews auth_time, never how long a session may last: the flow's
// lifetime runs from when the session began, and prompt=login is still a
// whole new sign-in.
func TestLiveAStepUpNeverStretchesASession(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "social", "totp")
	vault := f.requiringApp(t)
	f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+vault.flowID, map[string]any{"session_lifetime_hours": 1}, nil)

	owner := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(owner, challenge)
	secret, _ := addAuthenticator(t, owner)

	page(t, owner.visit(vault.authorizeURL(challenge, nil)), "/second-factor")
	if status, to := answerSecondFactor(owner, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}); status != http.StatusOK {
		t.Fatalf("the step-up = %d %s", status, to)
	}

	// prompt=login asks for everything again, the step-up notwithstanding.
	page(t, owner.visit(vault.authorizeURL(challenge, url.Values{"prompt": {"login"}})), "/login")

	// Half an hour into a one-hour session, a max_age of ten minutes is met
	// by the step-up a moment ago.
	f.ageSession(t, owner, 30*time.Minute)
	codeBack(t, owner.visit(vault.authorizeURL(challenge, url.Values{"max_age": {"600"}})), vaultRedirect)

	// Two hours in, the session is past the flow's lifetime, however
	// recently it passed a second factor.
	f.ageSession(t, owner, 90*time.Minute)
	page(t, owner.visit(vault.authorizeURL(challenge, nil)), "/login")
}

// A sign-in to an application that requires a second factor passes one, or
// sets one up when the account has none; an emailed code is not one.
func TestLiveARequiredSecondFactorIsAskedAtSignIn(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "email_code", "totp")
	vault := f.requiringApp(t)

	// A password and an emailed code: no mfa in the claim, and not enough
	// for the application that requires a second factor.
	owner := f.s.browser()
	verifier, challenge := pkce(t)
	login := owner.visit(f.authorizeURL(challenge, nil))
	var held struct {
		Code *struct {
			Handle string `json:"handle"`
		} `json:"code"`
	}
	owner.account(http.MethodPost, "/login", map[string]any{
		"request": login.Query().Get("request"), "email": adaEmail, "password": adaPassword,
	}, &held)
	if held.Code == nil {
		t.Fatal("the shop's sign-in did not wait for an emailed code")
	}
	var signedIn struct {
		RedirectTo string `json:"redirect_to"`
	}
	emailed := codeIn(t, f.s.mail.wait(t, adaEmail).Body, 6)
	if status := owner.account(http.MethodPost, "/login/code", map[string]any{"handle": held.Code.Handle, "code": emailed}, &signedIn); status != http.StatusOK {
		t.Fatalf("the emailed code = %d", status)
	}
	back, _ := url.Parse(signedIn.RedirectTo)
	shop := f.exchange(codeBack(t, back, shopRedirect), verifier)
	if got, want := amrOf(t, shop), []string{"pwd", "otp"}; !slices.Equal(got, want) {
		t.Errorf("a password and an emailed code: amr = %v, want %v", got, want)
	}

	// That session has passed no factor of the account's own, which has
	// none: the requiring application has one set up in the session.
	page(t, owner.visit(vault.authorizeURL(challenge, nil)), "/second-factor")
	var offer enrolOffer
	if owner.account(http.MethodGet, "/login/second-factor", nil, &offer); !offer.SecondFactor.Enrol || !offer.SecondFactor.StepUp {
		t.Errorf("the emailed-code session at the requiring application = %+v, want a factor set up in it", offer.SecondFactor)
	}

	// Signing in to it without a factor waits for one to be set up, before
	// any session.
	fresh := f.s.browser()
	at := fresh.visit(vault.authorizeURL(challenge, nil))
	var setUp struct {
		SecondFactor bool `json:"second_factor"`
	}
	fresh.signInAs(adaEmail, adaPassword, page(t, at, "/login"), false, &setUp)
	if !setUp.SecondFactor {
		t.Error("signing in without a factor did not wait for one to be set up")
	}
	if status := fresh.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Error("the sign-in waiting for a factor made a session")
	}

	// With a factor, the sign-in waits for it, then reaches the application.
	secret, _ := addAuthenticator(t, owner)
	withFactor := f.s.browser()
	verifier, challenge = pkce(t)
	at = withFactor.visit(vault.authorizeURL(challenge, nil))
	var waiting struct {
		SecondFactor bool `json:"second_factor"`
	}
	withFactor.account(http.MethodPost, "/login", map[string]any{
		"request": page(t, at, "/login"), "email": adaEmail, "password": adaPassword,
	}, &waiting)
	if !waiting.SecondFactor {
		t.Fatal("the sign-in did not wait for the second factor")
	}
	status, to := answerSecondFactor(withFactor, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))})
	if status != http.StatusOK {
		t.Fatalf("the second factor = %d %s", status, to)
	}
	back, _ = url.Parse(to)
	tokens := vault.exchange(codeBack(t, back, vaultRedirect), verifier)
	if got, want := amrOf(t, tokens), []string{"pwd", "otp", "mfa"}; !slices.Equal(got, want) {
		t.Errorf("a password and an authenticator: amr = %v, want %v", got, want)
	}
}
