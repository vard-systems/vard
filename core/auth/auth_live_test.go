package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"vard/core/database/databasetest"
	"vard/core/model"
	"vard/core/store"
	"vard/tools/jose"
)

const (
	liveUsername = "root"
	livePassword = "root-password-1"
)

// clock is a time the test moves by hand.
type clock struct{ at time.Time }

func (c *clock) now() time.Time          { return c.at }
func (c *clock) advance(d time.Duration) { c.at = c.at.Add(d) }

// liveService is a Service on a throwaway database holding one active
// administrator, on a clock the test moves. requireMFA is the panel's
// setting.
func liveService(t *testing.T, requireMFA bool) (*Service, *clock) {
	t.Helper()
	ctx := context.Background()

	db, _ := databasetest.Open(t)
	st := store.New(db)

	if err := st.EnsureAdminSecurity(ctx, requireMFA); err != nil {
		t.Fatal(err)
	}

	admin := model.Admin{Username: liveUsername, Email: "root@example.com", FirstName: "Root", Status: model.StatusActive}
	if err := admin.SetPassword(livePassword); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateAdmin(ctx, &admin); err != nil {
		t.Fatal(err)
	}

	sealer, err := jose.NewSealer("auth-live-test-secret-0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}

	// Whole seconds: Postgres keeps microseconds, and an expiry read back
	// must be the one written.
	c := &clock{at: time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC)}

	s := New(st, sealer, slog.New(slog.NewTextHandler(io.Discard, nil)), "test", nil)
	s.now = c.now

	return s, c
}

// A session ends at its lifetime and not a moment later, and which lifetime
// it gets depends on what the administrator still owes: the whole of one
// when signed in, a short one to enrol a second factor the panel requires.
func TestLiveSessionsEndAtTheirLifetime(t *testing.T) {
	tests := []struct {
		name       string
		requireMFA bool
		state      State
		lifetime   time.Duration
	}{
		{name: "a signed-in session", requireMFA: false, state: StateSignedIn, lifetime: SessionLifetime},
		{name: "a session owing a second factor to enrol", requireMFA: true, state: StateEnroll, lifetime: enrolmentLifetime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s, c := liveService(t, tt.requireMFA)

			// Nothing touches the session as the clock moves, so the longest
			// idle timeout keeps it from ending sooner than its lifetime.
			security, err := s.store.AdminSecurity(ctx)
			if err != nil {
				t.Fatal(err)
			}
			security.IdleTimeoutMinutes = model.MaxAdminIdleTimeout
			if err := s.store.SaveAdminSecurity(ctx, security); err != nil {
				t.Fatal(err)
			}

			token, _, state, err := s.Login(ctx, liveUsername, livePassword, Request{IP: "192.0.2.1"})
			if err != nil || state != tt.state {
				t.Fatalf("Login = %q, %v; want %q", state, err, tt.state)
			}

			c.advance(tt.lifetime - time.Second)
			if _, _, state, err := s.Session(ctx, token); err != nil || state != tt.state {
				t.Fatalf("a second before the end = %q, %v; want %q", state, err, tt.state)
			}

			c.advance(time.Second)
			if _, _, _, err := s.Session(ctx, token); !errors.Is(err, ErrNoSession) {
				t.Errorf("at the end = %v, want ErrNoSession", err)
			}
		})
	}
}

// Past the free wrong passwords an administrator waits, during which even the
// right one is refused with the same answer, and after which it works again.
func TestLiveAWaitEndsOnItsOwn(t *testing.T) {
	ctx := context.Background()
	s, c := liveService(t, false)
	req := Request{IP: "192.0.2.1"}

	for range model.ThrottleFree {
		if _, _, _, err := s.Login(ctx, liveUsername, "wrong-password", req); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("a wrong password = %v, want ErrInvalidCredentials", err)
		}
	}

	c.advance(model.ThrottleFirst - time.Second)
	if _, _, _, err := s.Login(ctx, liveUsername, livePassword, req); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("the right password while waiting = %v, want ErrInvalidCredentials", err)
	}

	c.advance(time.Second)
	if _, _, state, err := s.Login(ctx, liveUsername, livePassword, req); err != nil || state != StateSignedIn {
		t.Errorf("the right password once the wait ends = %q, %v; want signed in", state, err)
	}
}
