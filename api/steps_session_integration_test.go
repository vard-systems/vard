package api

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"vard/tools/totp"
)

// A session that the application's flow would not have made, such as one
// from before the flow was closed, must not finish that application's
// sign-in through the Consent or Agreements answer: only the session finish
// sent to the page may answer it.
func TestLiveAStepAnswerNeedsTheSessionThatReachedIt(t *testing.T) {
	type unmet struct {
		name string
		// before runs ahead of the first sign-in; after, once it has
		// finished, makes the flow one its session does not meet.
		before func(f *oauthFixture)
		after  func(f *oauthFixture, b *browser)
	}
	unmets := []unmet{
		{
			name: "a closed flow",
			after: func(f *oauthFixture, _ *browser) {
				f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+defaultFlow(t, f.super), map[string]any{
					"allow_sign_in": false,
				}, nil)
			},
		},
		{
			name: "an address the flow now requires proved",
			before: func(f *oauthFixture) {
				allowUnverifiedAddresses(t, f.super)
				if err := f.s.db.Exec("UPDATE users SET is_email_verified = false WHERE id = ?", f.userID).Error; err != nil {
					t.Fatal(err)
				}
			},
			after: func(f *oauthFixture, _ *browser) {
				f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+defaultFlow(t, f.super), map[string]any{
					"require_verified_email": true,
				}, nil)
			},
		},
		{
			name: "a session older than the flow's lifetime",
			after: func(f *oauthFixture, b *browser) {
				f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+defaultFlow(t, f.super), map[string]any{
					"session_lifetime_hours": 1,
				}, nil)
				f.ageSession(t, b, 2*time.Hour)
			},
		},
	}

	for _, condition := range unmets {
		for _, path := range []string{"/consent", "/agreements"} {
			t.Run(condition.name+" through "+path, func(t *testing.T) {
				f := newOAuthFixture(t)
				if condition.before != nil {
					condition.before(f)
				}
				b := f.s.browser()
				_, challenge := pkce(t)
				if code := f.signIn(b, challenge).Query().Get("code"); code == "" {
					t.Fatal("the first sign-in did not finish")
				}

				condition.after(f, b)

				verifier, challenge := pkce(t)
				request := page(t, b.visit(f.authorizeURL(challenge, nil)), "/login")

				if path == "/consent" {
					var problem problemBody
					if status := b.account(http.MethodGet, "/consent?request="+url.QueryEscape(request), nil, &problem); status != http.StatusGone || problem.Code != "request_expired" {
						t.Errorf("GET /consent = %d %+v, want request_expired", status, problem)
					}
				}

				var out struct {
					RedirectTo string `json:"redirect_to"`
					Code       string `json:"code"`
				}
				status := b.account(http.MethodPost, path, map[string]any{"request": request, "yes": true}, &out)
				at, _ := url.Parse(out.RedirectTo)
				if code := at.Query().Get("code"); code != "" {
					tokens := f.exchange(code, verifier)
					t.Errorf("the sign-in was finished through %s (%d): exchange = %d", path, status, tokens.status)
				}
				if status != http.StatusGone || out.Code != "request_expired" {
					t.Errorf("POST %s = %d %+v, want request_expired", path, status, out)
				}
			})
		}
	}
}

// The step page belongs to the session sent there: the same person's other
// session, which meets the flow, cannot answer it either.
func TestLiveAStepPageIsAnsweredOnlyByItsOwnSession(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "consent")

	b := f.s.browser()
	verifier, challenge := pkce(t)
	request := page(t, f.signIn(b, challenge), "/consent")

	other := f.s.browser()
	_, otherChallenge := pkce(t)
	page(t, f.signIn(other, otherChallenge), "/consent")

	if status := other.account(http.MethodPost, "/consent", map[string]any{"request": request, "yes": true}, nil); status != http.StatusGone {
		t.Errorf("another session answering the page = %d, want 410", status)
	}

	back := answer(t, b, "/consent", request, true)
	if tokens := f.exchange(back.Query().Get("code"), verifier); tokens.status != http.StatusOK {
		t.Fatalf("exchange after consent = %d %v", tokens.status, tokens.body)
	}
}

// A flow with the emailed-code step meets no existing session, yet a sign-in
// that has just passed the code goes on to the Consent page and finishes.
func TestLiveAnEmailedCodeThenConsentFinishes(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "email_code", "consent")

	b := f.s.browser()
	verifier, challenge := pkce(t)
	login := page(t, b.visit(f.authorizeURL(challenge, nil)), "/login")

	var held struct {
		Code *struct {
			Handle string `json:"handle"`
		} `json:"code"`
	}
	if status := b.account(http.MethodPost, "/login", map[string]string{"request": login, "email": adaEmail, "password": adaPassword}, &held); status != http.StatusOK || held.Code == nil {
		t.Fatalf("signing in = %d %+v, want a code to be asked for", status, held)
	}

	var out struct {
		RedirectTo string `json:"redirect_to"`
	}
	code := codeIn(t, f.s.mail.wait(t, adaEmail).Body, 6)
	if status := b.account(http.MethodPost, "/login/code", map[string]string{"handle": held.Code.Handle, "code": code}, &out); status != http.StatusOK {
		t.Fatalf("the emailed code = %d", status)
	}
	at, err := url.Parse(out.RedirectTo)
	if err != nil {
		t.Fatal(err)
	}

	back := answer(t, b, "/consent", page(t, at, "/consent"), true)
	if tokens := f.exchange(back.Query().Get("code"), verifier); tokens.status != http.StatusOK {
		t.Fatalf("exchange after consent = %d %v", tokens.status, tokens.body)
	}
}

// A step-up keeps the session's id, so the Consent page it leads to is
// answered by the session that stepped up, under its new cookie.
func TestLiveAStepUpThenConsentFinishes(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)
	f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+vault.flowID, map[string]any{
		"steps": []string{"identifier", "password", "totp", "consent"},
	}, nil)

	owner := f.s.browser()
	_, challenge := pkce(t)
	codeBack(t, f.signIn(owner, challenge), shopRedirect)
	secret, _ := addAuthenticator(t, owner)

	verifier, challenge := pkce(t)
	page(t, owner.visit(vault.authorizeURL(challenge, nil)), "/second-factor")

	status, to := answerSecondFactor(owner, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))})
	if status != http.StatusOK {
		t.Fatalf("the step-up = %d %s", status, to)
	}
	at, err := url.Parse(to)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(at.String(), testAccountURL+"/consent?") {
		t.Fatalf("the step-up went to %s, want the Consent page", at)
	}

	back := answer(t, owner, "/consent", page(t, at, "/consent"), true)
	if tokens := vault.exchange(codeBack(t, back, vaultRedirect), verifier); tokens.status != http.StatusOK {
		t.Fatalf("exchange after the step-up and consent = %d %v", tokens.status, tokens.body)
	}
}
