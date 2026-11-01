package api

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// An address is one account however it is typed: stored lower case, found by
// the unique index whatever the case of the sign-in, and refused a second
// time in other capitals.
func TestLiveAddressesAreOneAccountInAnyCase(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	var created struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": " Ada.Lovelace@Example.COM ", "first_name": "Ada", "is_email_verified": true,
		"password": "ada-password-1", "confirm_password": "ada-password-1",
	}, &created)

	if created.User.Email != "ada.lovelace@example.com" {
		t.Errorf("stored as %q, want it trimmed and lower case", created.User.Email)
	}

	if status := super.do(http.MethodPost, "/users", map[string]any{
		"email": "ada.lovelace@example.com", "password": "ada-password-2", "confirm_password": "ada-password-2",
	}, nil); status != http.StatusConflict {
		t.Errorf("the same address in other capitals = %d, want 409", status)
	}

	b := s.browser()
	if status := b.account(http.MethodPost, "/login", map[string]string{
		"email": "ADA.LOVELACE@example.com", "password": "ada-password-1",
	}, nil); status != http.StatusOK {
		t.Fatalf("signing in with the address in capitals = %d, want 200", status)
	}
	if email, _, _ := b.me(); email != "ada.lovelace@example.com" {
		t.Errorf("signed in as %q", email)
	}
}

// The sweep removes what has expired and nothing that has not: a session in
// use survives one, and is gone once its time is up — as is the activity log
// past its retention.
func TestLiveSweepRemovesOnlyWhatExpired(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	ctx := context.Background()

	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "grace@example.com", "is_email_verified": true,
		"password": "grace-password-1", "confirm_password": "grace-password-1",
	}, nil)

	b := s.browser()
	b.account(http.MethodPost, "/login", map[string]string{"email": "grace@example.com", "password": "grace-password-1"}, nil)

	now := time.Now()
	if _, err := s.store.Sweep(ctx, now, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if status := b.account(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Fatalf("after a sweep, a session in use = %d, want it kept", status)
	}

	var logs struct {
		Logs []struct{} `json:"logs"`
	}
	super.must(http.StatusOK, http.MethodGet, "/logs?limit=100", nil, &logs)
	if len(logs.Logs) == 0 {
		t.Fatal("the activity log is empty before the sweep that should empty it")
	}

	// A year and more on: the session has expired, and so has everything the
	// log held a retention ago.
	later := now.Add(400 * 24 * time.Hour)
	removed, err := s.store.Sweep(ctx, later, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if removed == 0 {
		t.Error("a sweep a year on removed nothing")
	}

	if status := b.account(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("a swept session = %d, want 401", status)
	}

	// The administrator's own session has gone the same way, so the log is
	// read from the store.
	left, err := s.store.AuditLog(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Errorf("%d log entries survived their retention", len(left))
	}
}
