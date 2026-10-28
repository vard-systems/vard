package api

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// A login flow with the emailed code step holds a sign-in back: the password
// alone makes no session, the code in the message does, and a code that is
// wrong, spent or somebody else's does not.
func TestLiveEmailedCodeFinishesASignIn(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	allowUnverifiedAddresses(t, super)

	// Six digits is the default; four makes the codes in this test shorter to
	// read, and proves the setting is the one the server actually uses.
	super.must(http.StatusOK, http.MethodPatch, "/otp", map[string]any{
		"code_length": 4, "max_attempts": 2, "resend_seconds": 0,
	}, nil)

	flow := defaultFlow(t, super)
	super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+flow, map[string]any{
		"steps": []string{"identifier", "password", "email_code"},
	}, nil)

	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "ada@example.com", "password": "ada-password-1", "confirm_password": "ada-password-1",
	}, nil)

	type challenge struct {
		Handle       string `json:"handle"`
		Email        string `json:"email"`
		AttemptsLeft int    `json:"attempts_left"`
	}
	type signedIn struct {
		Code *challenge `json:"code"`
	}

	b := s.browser()

	var held signedIn
	if status := b.account(http.MethodPost, "/login", map[string]string{
		"email": "ada@example.com", "password": "ada-password-1",
	}, &held); status != http.StatusOK {
		t.Fatalf("signing in = %d, want 200", status)
	}
	if held.Code == nil {
		t.Fatal("the password alone finished the sign-in; the flow asks for a code")
	}
	if held.Code.Handle == "" {
		t.Fatal("the sign-in was held with no handle to come back with")
	}

	// The address is shown back so somebody knows where to look, with the
	// middle of it hidden.
	if !strings.HasSuffix(held.Code.Email, "@example.com") || strings.Contains(held.Code.Email, "ada@") {
		t.Errorf("the address shown = %q, want it masked", held.Code.Email)
	}

	// Nobody is signed in until the code is typed.
	if status := b.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Error("a session was made before the code was typed")
	}

	code := codeIn(t, s.mail.wait(t, "ada@example.com").Body, 4)

	// A wrong code is refused with guesses left, then the sign-in is over.
	var wrong problemBody
	if status := b.account(http.MethodPost, "/login/code", map[string]string{
		"handle": held.Code.Handle, "code": "0000",
	}, &wrong); status != http.StatusBadRequest || wrong.Code != "code_invalid" {
		t.Errorf("a wrong code = %d %+v, want code_invalid", status, wrong)
	}

	// The handle is the secret, so the right code under a handle nobody has
	// is worth nothing.
	var stolen problemBody
	if status := b.account(http.MethodPost, "/login/code", map[string]string{
		"handle": "not-a-handle", "code": code,
	}, &stolen); status != http.StatusGone || stolen.Code != "code_expired" {
		t.Errorf("a code under an unknown handle = %d %+v, want code_expired", status, stolen)
	}

	if status := b.account(http.MethodPost, "/login/code", map[string]string{
		"handle": held.Code.Handle, "code": code,
	}, nil); status != http.StatusOK {
		t.Fatalf("the code that was sent = %d, want 200", status)
	}

	if status := b.account(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Error("the right code did not sign Ada in")
	}

	// The code is spent: the same one again finishes nothing.
	fresh := s.browser()
	var spent problemBody
	if status := fresh.account(http.MethodPost, "/login/code", map[string]string{
		"handle": held.Code.Handle, "code": code,
	}, &spent); status != http.StatusGone || spent.Code != "code_expired" {
		t.Errorf("a code used twice = %d %+v, want code_expired", status, spent)
	}
}

// A sign-in is over once the guesses run out, and the code it was waiting for
// stops working with it.
func TestLiveEmailedCodeRunsOutOfGuesses(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	allowUnverifiedAddresses(t, super)

	super.must(http.StatusOK, http.MethodPatch, "/otp", map[string]any{
		"code_length": 4, "max_attempts": 1,
	}, nil)

	flow := defaultFlow(t, super)
	super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+flow, map[string]any{
		"steps": []string{"identifier", "password", "email_code"},
	}, nil)

	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "grace@example.com", "password": "grace-password-1", "confirm_password": "grace-password-1",
	}, nil)

	var held struct {
		Code *struct {
			Handle string `json:"handle"`
		} `json:"code"`
	}
	b := s.browser()
	b.account(http.MethodPost, "/login", map[string]string{
		"email": "grace@example.com", "password": "grace-password-1",
	}, &held)
	if held.Code == nil {
		t.Fatal("the sign-in was not held for a code")
	}

	code := codeIn(t, s.mail.wait(t, "grace@example.com").Body, 4)

	var used problemBody
	if status := b.account(http.MethodPost, "/login/code", map[string]string{
		"handle": held.Code.Handle, "code": "0000",
	}, &used); status != http.StatusGone || used.Code != "code_attempts_used" {
		t.Fatalf("the last guess = %d %+v, want code_attempts_used", status, used)
	}

	// Even the code that was sent: the sign-in it belonged to is over.
	var over problemBody
	if status := b.account(http.MethodPost, "/login/code", map[string]string{
		"handle": held.Code.Handle, "code": code,
	}, &over); status != http.StatusGone || over.Code != "code_expired" {
		t.Errorf("the right code after the guesses ran out = %d %+v, want code_expired", status, over)
	}
}

// codeIn reads the one-time code out of a message: the only run of exactly
// that many digits in it.
func codeIn(t *testing.T, body string, length int) string {
	t.Helper()

	matches := regexp.MustCompile(`\b\d{`+strconv.Itoa(length)+`}\b`).FindAllString(body, -1)
	if len(matches) != 1 {
		t.Fatalf("found %d codes in the message, want one:\n%s", len(matches), body)
	}

	return matches[0]
}
