package oidc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"vard/internal/jose"
	"vard/internal/model"
)

// askAnswer is a challenge hook wanting the answer "right".
type askAnswer struct{ asked int }

func (a *askAnswer) Check(_ context.Context, attempt ChallengeAttempt) error {
	a.asked++
	if attempt.Answer != "right" {
		return ErrChallengeRequired
	}
	return nil
}

// An installation's challenge hook is asked only once a sign-in's keys have
// failed ChallengeAfter times; until it is satisfied the password is not
// looked at, and with its answer the sign-in goes on.
func TestLiveTheChallengeHookIsAskedAfterFailures(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	sealer, err := jose.NewSealer("challenge-test-secret-key-0123456789")
	if err != nil {
		t.Fatal(err)
	}
	hook := &askAnswer{}
	s := &Service{store: st, sealer: sealer, log: slog.New(slog.NewTextHandler(io.Discard, nil)), now: time.Now, challenge: hook}

	if _, err := st.DefaultLoginFlow(ctx); err != nil {
		t.Fatal(err)
	}
	user := &model.User{Email: "ada@example.com", IsActive: true, IsEmailVerified: true, Data: map[string]any{}}
	if err := user.SetPassword("ada-password-1"); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}

	client := Client{IP: "198.51.100.1"}
	for range ChallengeAfter {
		if _, err := s.SignIn(ctx, user.Email, "wrong", "", false, client); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("a wrong password = %v", err)
		}
	}
	if hook.asked != 0 {
		t.Errorf("the hook was asked %d times before %d failures", hook.asked, ChallengeAfter)
	}

	if _, err := s.SignIn(ctx, user.Email, "ada-password-1", "", false, client); !errors.Is(err, ErrChallengeRequired) {
		t.Errorf("the right password without the answer = %v, want ErrChallengeRequired", err)
	}

	client.Challenge = "right"
	if result, err := s.SignIn(ctx, user.Email, "ada-password-1", "", false, client); err != nil || result.Session == nil {
		t.Errorf("with the answer = %+v %v, want signed in", result, err)
	}
}
