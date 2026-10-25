package oidc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"vard/internal/model"
	"vard/internal/store"
)

// codesAtOnce is how many codes each case types together: well past the
// guesses a sign-in is allowed, and about what the rate limit lets one
// address send in a burst.
const codesAtOnce = 20

// A sign-in waiting for its emailed code allows a few guesses, and codes typed
// at the same moment are held to that as surely as codes typed one after
// another: each is counted, and no more of them are compared than the sign-in
// had guesses left. Counted from the row each request had loaded, a burst of
// wrong codes would cost one guess, and a six-digit code could be walked
// through twenty at a time.
func TestLiveCodesTypedAtOnceAreEachCounted(t *testing.T) {
	tests := []struct {
		name string
		// typed is the code the i-th request sends.
		typed func(right string, i int) string
		// sessions is how many sessions the burst should make.
		sessions int
		// spent says the sign-in should have no guesses left afterwards, so
		// even the right code is refused.
		spent bool
	}{
		{
			name:     "wrong codes typed together use up the guesses",
			typed:    func(right string, i int) string { return wrongCode(right, i) },
			sessions: 0,
			spent:    true,
		},
		{
			name:     "the right code typed in several tabs signs in once",
			typed:    func(right string, _ int) string { return right },
			sessions: 1,
			spent:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			st := testStore(t)
			s := &Service{store: st, log: slog.New(slog.NewTextHandler(io.Discard, nil)), now: time.Now}

			settings, err := st.OTPSettings(ctx)
			if err != nil {
				t.Fatal(err)
			}
			// Made before the burst, so the one request that signs in does
			// not race the others to create it.
			if _, err := st.DefaultLoginFlow(ctx); err != nil {
				t.Fatal(err)
			}

			user, handle, right := waitingSignIn(t, st, settings)

			results := make([]error, codesAtOnce)
			var start, done sync.WaitGroup
			start.Add(1)
			for i := range codesAtOnce {
				done.Go(func() {
					start.Wait()
					_, results[i] = s.SubmitLoginCode(ctx, handle, tt.typed(right, i), Client{})
				})
			}
			start.Done()
			done.Wait()

			for _, err := range results {
				if err != nil && !errors.Is(err, ErrCodeInvalid) && !errors.Is(err, ErrCodeAttemptsUsed) && !errors.Is(err, ErrCodeExpired) {
					t.Fatalf("a code was answered with %v", err)
				}
			}

			sessions, err := st.ActiveUserSessions(ctx, user.ID, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if len(sessions) != tt.sessions {
				t.Errorf("%d sessions were made, want %d", len(sessions), tt.sessions)
			}

			code, err := st.LoginCodeByHash(ctx, model.HashSecret(handle))
			if err != nil {
				t.Fatal(err)
			}
			if code.Attempts > settings.MaxAttempts {
				t.Errorf("%d codes were counted, more than the %d allowed", code.Attempts, settings.MaxAttempts)
			}

			if !tt.spent {
				return
			}

			if code.Attempts != settings.MaxAttempts {
				t.Errorf("%d wrong codes cost %d guesses, want all %d", codesAtOnce, code.Attempts, settings.MaxAttempts)
			}

			if _, err := s.SubmitLoginCode(ctx, handle, right, Client{}); err == nil {
				t.Error("the right code signed in after the guesses were used up")
			}
		})
	}
}

// waitingSignIn is a user and a sign-in of theirs waiting for a code: the
// handle the page would hold, and the code the message would carry.
func waitingSignIn(t *testing.T, st *store.Store, settings *model.OTPSettings) (*model.User, string, string) {
	t.Helper()
	ctx := context.Background()

	user := &model.User{Email: "codes@example.com", IsActive: true, IsEmailVerified: true, Data: map[string]any{}}
	if err := st.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}

	handle, handleHash, err := model.NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	code, codeHash, err := model.NewOTP(settings.CodeLength)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	waiting := &model.LoginCode{
		HandleHash: handleHash,
		CodeHash:   codeHash,
		UserID:     user.ID,
		SentAt:     now,
		ExpiresAt:  now.Add(settings.Lifetime()),
	}
	if err := st.CreateLoginCode(ctx, waiting); err != nil {
		t.Fatal(err)
	}

	return user, handle, code
}

// wrongCode is a code of the right length that is not `right`, a different
// one for each i.
func wrongCode(right string, i int) string {
	b := []byte(right)
	b[0] = '0' + (right[0]-'0'+1)%10
	b[len(b)-1] = '0' + byte(i%10)
	return string(b)
}
