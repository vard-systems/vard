package api

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// withSteps sets the default flow's steps.
func (f *oauthFixture) withSteps(steps ...string) {
	f.s.t.Helper()
	f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+defaultFlow(f.s.t, f.super), map[string]any{
		"steps": steps,
	}, nil)
}

// allowUnverifiedAddresses lets the default flow sign in an address nobody
// has proved, as an installation from before that was the default keeps
// doing until an administrator turns it on.
func allowUnverifiedAddresses(t *testing.T, super *client) {
	t.Helper()
	super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+defaultFlow(t, super), map[string]any{
		"require_verified_email": false,
	}, nil)
}

// page asserts the browser was sent to a sign-in page of the id app and
// returns the request it names.
func page(t *testing.T, at *url.URL, path string) string {
	t.Helper()

	if !strings.HasPrefix(at.String(), testAccountURL+path+"?") {
		t.Fatalf("the browser went to %s, want %s", at, path)
	}

	return at.Query().Get("request")
}

// answer posts the Agreements or Consent page's answer and returns where the
// browser goes.
func answer(t *testing.T, b *browser, path, request string, yes bool) *url.URL {
	t.Helper()

	var out struct {
		RedirectTo string `json:"redirect_to"`
		Code       string `json:"code"`
	}
	if status := b.account(http.MethodPost, path, map[string]any{"request": request, "yes": yes}, &out); status != http.StatusOK {
		t.Fatalf("POST %s = %d %+v", path, status, out)
	}

	at, err := url.Parse(out.RedirectTo)
	if err != nil {
		t.Fatal(err)
	}

	return at
}

// The Consent step asks once for what an application wants, lets the user
// say no, asks again when the application wants more or sends
// prompt=consent, and never asks a silent sign-in.
func TestLiveConsent(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "consent")

	b := f.s.browser()
	verifier, challenge := pkce(t)
	request := page(t, f.signIn(b, challenge), "/consent")

	var listed struct {
		Consent struct {
			Application struct {
				Name string `json:"name"`
			} `json:"application"`
			Scopes []struct {
				Name string `json:"name"`
				API  string `json:"api"`
			} `json:"scopes"`
		} `json:"consent"`
	}
	if status := b.account(http.MethodGet, "/consent?request="+url.QueryEscape(request), nil, &listed); status != http.StatusOK {
		t.Fatalf("GET /consent = %d", status)
	}
	if listed.Consent.Application.Name != "Shop" || len(listed.Consent.Scopes) != 7 {
		t.Fatalf("the consent page lists %+v, want Shop and its seven scopes", listed.Consent)
	}
	if last := listed.Consent.Scopes[6]; last.Name != "orders:write" || last.API != "Orders" {
		t.Errorf("an API scope is listed as %+v, want orders:write of Orders", last)
	}

	// A silent sign-in from the same browser is told, not shown a page.
	silent := b.visit(f.authorizeURL(challenge, url.Values{"prompt": {"none"}}))
	if silent.Query().Get("error") != "consent_required" {
		t.Errorf("prompt=none before consent led to %s, want consent_required", silent)
	}

	back := answer(t, b, "/consent", request, true)
	if tokens := f.exchange(back.Query().Get("code"), verifier); tokens.status != http.StatusOK {
		t.Fatalf("exchange after consent = %d %v", tokens.status, tokens.body)
	}

	// Allowed once, the same request, or a narrower one, goes straight through.
	for _, extra := range []url.Values{nil, {"scope": {"openid email"}, "audience": {""}}} {
		again := f.s.browser()
		_, challenge := pkce(t)
		if code := f.signInWith(again, challenge, extra).Query().Get("code"); code == "" {
			t.Errorf("a sign-in asking for %v after consent was not let through", extra)
		}
	}

	// prompt=consent asks regardless, and allowing it does not ask again.
	forced := f.s.browser()
	_, challenge = pkce(t)
	request = page(t, f.signInWith(forced, challenge, url.Values{"prompt": {"consent"}}), "/consent")
	if code := answer(t, forced, "/consent", request, true).Query().Get("code"); code == "" {
		t.Error("allowing a prompt=consent sign-in did not finish it")
	}

	// Disconnecting the application forgets what was allowed.
	if status := b.account(http.MethodDelete, "/connected-applications/"+f.clientID, nil, nil); status != http.StatusNoContent {
		t.Fatalf("disconnecting = %d", status)
	}
	_, challenge = pkce(t)
	page(t, f.signIn(f.s.browser(), challenge), "/consent")

	// Saying no sends the browser back refused, and the request is spent.
	refuser := f.s.browser()
	_, challenge = pkce(t)
	request = page(t, f.signIn(refuser, challenge), "/consent")
	refused := answer(t, refuser, "/consent", request, false)
	if refused.Query().Get("error") != "access_denied" || !strings.HasPrefix(refused.String(), shopRedirect) {
		t.Errorf("refusing led to %s, want access_denied at the application", refused)
	}
	if status := refuser.account(http.MethodPost, "/consent", map[string]any{"request": request, "yes": true}, nil); status != http.StatusGone {
		t.Errorf("answering a refused request again = %d, want 410", status)
	}
}

// The Agreements step asks for the organization's terms once, again after
// they change, and not at all when there are none.
func TestLiveAgreements(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "terms")

	signIn := func() (*browser, *url.URL) {
		b := f.s.browser()
		_, challenge := pkce(t)
		return b, f.signIn(b, challenge)
	}

	// No links, nothing to accept.
	if _, back := signIn(); back.Query().Get("code") == "" {
		t.Fatalf("with no terms the sign-in went to %s, want the application", back)
	}

	f.super.must(http.StatusOK, http.MethodPatch, "/organization", map[string]any{"terms_url": "https://example.com/terms"}, nil)

	b, at := signIn()
	request := page(t, at, "/agreements")
	declined := answer(t, b, "/agreements", request, false)
	if declined.Query().Get("error") != "access_denied" {
		t.Errorf("declining led to %s, want access_denied", declined)
	}

	b, at = signIn()
	if code := answer(t, b, "/agreements", page(t, at, "/agreements"), true).Query().Get("code"); code == "" {
		t.Fatal("accepting did not finish the sign-in")
	}

	if _, back := signIn(); back.Query().Get("code") == "" {
		t.Errorf("accepted once, the next sign-in went to %s", back)
	}

	// New terms are asked for again.
	f.super.must(http.StatusOK, http.MethodPatch, "/organization", map[string]any{"privacy_url": "https://example.com/privacy"}, nil)
	_, at = signIn()
	page(t, at, "/agreements")
}

// Both steps in one flow come in order: the terms first, then the
// application.
func TestLiveAgreementsThenConsent(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "terms", "consent")
	f.super.must(http.StatusOK, http.MethodPatch, "/organization", map[string]any{"terms_url": "https://example.com/terms"}, nil)

	b := f.s.browser()
	_, challenge := pkce(t)
	request := page(t, f.signIn(b, challenge), "/agreements")
	request = page(t, answer(t, b, "/agreements", request, true), "/consent")
	if code := answer(t, b, "/consent", request, true).Query().Get("code"); code == "" {
		t.Error("passing both steps did not finish the sign-in")
	}
}

// The registration form's checkbox shows the organization's links when the
// application has none of its own; ticked, it is the Agreements step passed,
// so a new account is not asked twice.
func TestLiveRegistrationAcceptsTheAgreements(t *testing.T) {
	f := newOAuthFixture(t)
	allowUnverifiedAddresses(t, f.super)
	f.withSteps("identifier", "password", "terms")
	f.super.must(http.StatusOK, http.MethodPatch, "/organization", map[string]any{"terms_url": "https://example.com/terms"}, nil)
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

	var out struct {
		RedirectTo string `json:"redirect_to"`
	}
	if status := b.account(http.MethodPost, "/register", map[string]any{
		"request": login.Query().Get("request"), "email": "grace@example.com",
		"password": "grace-password-1", "accept_terms": true,
	}, &out); status != http.StatusOK {
		t.Fatalf("registering = %d", status)
	}
	if back, _ := url.Parse(out.RedirectTo); back == nil || back.Query().Get("code") == "" {
		t.Errorf("a new account that ticked the terms went to %s, want the application", out.RedirectTo)
	}
}

// An application the user allowed shows among their connected applications
// even when it holds no refresh token, and disconnecting it there takes the
// consent back.
func TestLiveAllowedApplicationsCanBeDisconnected(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "consent")

	// Without offline_access: nothing is issued that would outlive the
	// sign-in, so only the consent connects them.
	narrow := url.Values{"scope": {"openid email"}, "audience": {""}}

	b := f.s.browser()
	verifier, challenge := pkce(t)
	request := page(t, f.signInWith(b, challenge, narrow), "/consent")
	back := answer(t, b, "/consent", request, true)
	if tokens := f.exchange(back.Query().Get("code"), verifier); tokens.str("refresh_token") != "" {
		t.Fatal("a refresh token was issued without offline_access")
	}

	var listed struct {
		Applications []struct {
			ClientID string   `json:"client_id"`
			Scopes   []string `json:"scopes"`
		} `json:"applications"`
	}
	b.account(http.MethodGet, "/connected-applications", nil, &listed)
	if len(listed.Applications) != 1 || listed.Applications[0].ClientID != f.clientID {
		t.Fatalf("connected applications = %+v, want the one allowed", listed.Applications)
	}

	if status := b.account(http.MethodDelete, "/connected-applications/"+f.clientID, nil, nil); status != http.StatusNoContent {
		t.Fatalf("disconnecting an allowed application = %d, want 204", status)
	}

	_, challenge = pkce(t)
	page(t, f.signInWith(f.s.browser(), challenge, narrow), "/consent")
}
