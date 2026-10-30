package api

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// A login flow decides how people sign in, not only what the page offers: a
// flow without a password refuses one, a flow that requires a verified
// address sends a link instead of a session, and a session lasts as long as
// the flow says.
func TestLiveLoginFlowsDecideSignIn(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	var flows struct {
		Flows []struct {
			ID        string `json:"id"`
			IsDefault bool   `json:"is_default"`
		} `json:"flows"`
	}
	super.must(http.StatusOK, http.MethodGet, "/login-flows", nil, &flows)
	var flow string
	for _, f := range flows.Flows {
		if f.IsDefault {
			flow = f.ID
		}
	}
	if flow == "" {
		t.Fatal("there is no default flow")
	}
	change := func(body map[string]any) {
		super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+flow, body, nil)
	}

	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "ada@example.com", "password": "ada-password-1", "confirm_password": "ada-password-1",
	}, nil)
	signIn := func(b *browser) (int, problemBody) {
		var out problemBody
		status := b.account(http.MethodPost, "/login", map[string]string{"email": "ada@example.com", "password": "ada-password-1"}, &out)
		return status, out
	}

	// Without a password step, a password is refused before it is checked,
	// and nobody is sent a reset link.
	change(map[string]any{"steps": []string{"identifier", "social"}})
	if status, out := signIn(s.browser()); status != http.StatusForbidden || out.Code != "password_not_offered" {
		t.Errorf("a password at a flow without one = %d %+v, want password_not_offered", status, out)
	}
	s.browser().account(http.MethodPost, "/forgot-password", map[string]string{"email": "ada@example.com"}, nil)

	// A flow that requires a verified address sends a link, not a session.
	change(map[string]any{"steps": []string{"identifier", "password"}, "require_verified_email": true, "session_lifetime_hours": 2})
	b := s.browser()
	if status, out := signIn(b); status != http.StatusForbidden || out.Code != "email_not_verified" {
		t.Fatalf("an unverified address = %d %+v, want email_not_verified", status, out)
	}

	sent := s.mail.wait(t, "ada@example.com")
	if strings.Contains(sent.Subject, "password") {
		t.Fatalf("the first email to Ada is %q: the password-less flow sent a reset link", sent.Subject)
	}
	token := tokenIn(t, sent.Body, "/verify-email")

	if status := b.account(http.MethodPost, "/verify-email", map[string]string{"token": token}, nil); status != http.StatusOK {
		t.Fatalf("verifying the address = %d, want 200", status)
	}
	var reused problemBody
	if status := b.account(http.MethodPost, "/verify-email", map[string]string{"token": token}, &reused); status != http.StatusGone || reused.Code != "verification_invalid" {
		t.Errorf("a verification link used twice = %d %+v, want verification_invalid", status, reused)
	}

	// Verified, Ada signs in — for as long as the flow says.
	if status, out := signIn(b); status != http.StatusOK {
		t.Fatalf("signing in once verified = %d %+v", status, out)
	}

	var sessions struct {
		Sessions []struct {
			SignedInAt time.Time `json:"signed_in_at"`
			ExpiresAt  time.Time `json:"expires_at"`
		} `json:"sessions"`
	}
	super.must(http.StatusOK, http.MethodGet, "/user-sessions?search=ada", nil, &sessions)
	if len(sessions.Sessions) != 1 {
		t.Fatalf("Ada's sessions = %+v, want one", sessions.Sessions)
	}
	if lasts := sessions.Sessions[0].ExpiresAt.Sub(sessions.Sessions[0].SignedInAt); lasts != 2*time.Hour {
		t.Errorf("the session lasts %v, want the flow's two hours", lasts)
	}

	// A flow cannot be saved that nobody could get through today.
	var refused problemBody
	if status := super.do(http.MethodPatch, "/login-flows/"+flow, map[string]any{"steps": []string{"identifier", "totp"}}, &refused); status != http.StatusBadRequest {
		t.Errorf("a flow of steps not run yet = %d %+v, want 400", status, refused)
	}
}
