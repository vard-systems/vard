package api

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"vard/internal/config"
	"vard/internal/totp"
)

// wrongCodeFor is a six-digit code no step the server accepts would give.
func wrongCodeFor(t *testing.T, secret string) string {
	t.Helper()

	accepted := []string{}
	for step := -totp.Skew - 1; step <= totp.Skew+1; step++ {
		accepted = append(accepted, codeFor(t, secret, time.Now().Add(time.Duration(step)*totp.Period)))
	}

	for i := 0; ; i++ {
		if code := fmt.Sprintf("%06d", i); !slices.Contains(accepted, code) {
			return code
		}
	}
}

// Someone with an administrator's password cannot keep guessing codes by
// signing in again between guesses: the password does not give back what the
// guesses took, so the account locks.
func TestLiveAdminCodeGuessesReachTheLockout(t *testing.T) {
	s := newLiveServerWith(t, func(cfg *config.Config) { cfg.AdminMFARequired = true })

	root := s.client()
	root.must(http.StatusCreated, http.MethodPost, "/setup", map[string]string{
		"email": superEmail, "password": superPassword, "first_name": "Root",
		"setup_token": s.setupToken,
	}, nil)
	root.loginAnswer(superEmail, superPassword)
	var begun struct {
		Enrolment struct {
			Secret string `json:"secret"`
		} `json:"enrolment"`
	}
	root.must(http.StatusOK, http.MethodPost, "/mfa/totp", nil, &begun)
	secret := begun.Enrolment.Secret
	var confirmed struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	root.must(http.StatusOK, http.MethodPost, "/mfa/totp/confirm", map[string]string{"code": codeFor(t, secret, time.Now())}, &confirmed)

	// Signing in properly, again and again, never locks: each finished
	// sign-in gives back what its password and code took.
	for i := range 6 {
		laptop := s.client()
		if _, answer := laptop.loginAnswer(superEmail, superPassword); answer.Next != "mfa" {
			t.Fatalf("sign-in %d next = %q, want mfa", i+1, answer.Next)
		}
		laptop.must(http.StatusOK, http.MethodPost, "/auth/mfa", map[string]string{"code": confirmed.RecoveryCodes[i]}, nil)
	}

	wrong := wrongCodeFor(t, secret)
	attacker := s.client()
	for range 3 {
		attacker.loginAnswer(superEmail, superPassword)
		for range 4 {
			attacker.do(http.MethodPost, "/auth/mfa", map[string]string{"code": wrong}, nil)
		}
	}

	if status, _ := attacker.loginAnswer(superEmail, superPassword); status == http.StatusOK {
		t.Error("the password still signs in after a dozen wrong codes, want the account locked")
	}
	next := codeFor(t, secret, time.Now().Add(totp.Period))
	if status := attacker.do(http.MethodPost, "/auth/mfa", map[string]string{"code": next}, nil); status == http.StatusOK {
		t.Error("the right code still signed in after a dozen wrong ones, want the account locked")
	}
}

// A session left open cannot be used to find the user's password: guesses at
// the current one count like sign-in guesses, and once they have run out the
// right one is refused too.
func TestLiveUserCurrentPasswordGuessesLock(t *testing.T) {
	f := newOAuthFixture(t)

	laptop := f.s.browser()
	if status := laptop.account(http.MethodPost, "/login", map[string]string{"email": adaEmail, "password": adaPassword}, nil); status != http.StatusOK {
		t.Fatalf("login = %d", status)
	}

	// A right password gives its attempt back, so changing the password the
	// honest way does not count against anyone.
	for range 6 {
		var refused problemBody
		laptop.account(http.MethodPost, "/password", map[string]string{"current_password": adaPassword, "new_password": adaPassword}, &refused)
		if refused.Code != "password_unchanged" {
			t.Fatalf("the right password, unchanged = %+v, want password_unchanged", refused)
		}
	}

	for i := range 5 {
		laptop.account(http.MethodPost, "/password", map[string]string{"current_password": fmt.Sprintf("guess-%d-password", i), "new_password": "irrelevant-123"}, nil)
	}

	if status := laptop.account(http.MethodPost, "/password", map[string]string{"current_password": adaPassword, "new_password": "taken-over-password"}, nil); status != http.StatusBadRequest {
		t.Errorf("the right current password after five wrong ones = %d, want 400", status)
	}

}

// The same holds for an administrator's own password.
func TestLiveAdminCurrentPasswordGuessesLock(t *testing.T) {
	s := newLiveServer(t)
	root := s.superAdmin()

	for i := range 5 {
		root.do(http.MethodPost, "/me/password", map[string]string{"current_password": fmt.Sprintf("guess-%d-password", i), "new_password": "irrelevant-password-1"}, nil)
	}

	if status := root.do(http.MethodPost, "/me/password", map[string]string{"current_password": superPassword, "new_password": "taken-over-password-1"}, nil); status != http.StatusBadRequest {
		t.Errorf("the right current password on a locked account = %d, want 400", status)
	}
	if status := s.client().login(superEmail, superPassword); status == http.StatusOK {
		t.Error("signing in after five wrong current passwords = 200, want the account locked")
	}
}
