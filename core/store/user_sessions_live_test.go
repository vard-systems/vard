package store

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"vard/core/cache"
	"vard/core/cache/cachetest"
	"vard/core/database/databasetest"
	"vard/core/model"
)

// A step-up is one conditional UPDATE: a session that ended after it was read
// (it expired, was signed out, or another step-up already rotated it) is not
// brought back under a new token, in the database or the cache.
func TestLiveAStepUpNeverRevivesAnEndedSession(t *testing.T) {
	db, _ := databasetest.Open(t)
	redis := cachetest.Open(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	tests := []struct {
		name string
		// end happens between the session being read and the step-up, and
		// returns when the step-up is tried.
		end func(st *Store, session *model.UserSession) time.Time
	}{
		{"it expired", func(_ *Store, session *model.UserSession) time.Time {
			return session.ExpiresAt.Add(time.Second)
		}},
		{"it expires the very moment of the step-up", func(_ *Store, session *model.UserSession) time.Time {
			return session.ExpiresAt
		}},
		{"it was signed out", func(st *Store, session *model.UserSession) time.Time {
			if err := st.RevokeUserSession(ctx, session.ID, now); err != nil {
				t.Fatal(err)
			}
			return now.Add(time.Second)
		}},
		{"another step-up rotated it first", func(st *Store, session *model.UserSession) time.Time {
			first := *session
			ok, err := st.StepUpUserSession(ctx, &first, newHash(t), model.WithSecondFactor(first.Methods, model.AMROneTimeCode), now)
			if err != nil || !ok {
				t.Fatalf("the first step-up: %v, %v", ok, err)
			}
			return now.Add(time.Second)
		}},
	}

	caches := []struct {
		name  string
		redis *cache.Redis
	}{{"database only", nil}}
	if redis != nil {
		caches = append(caches, struct {
			name  string
			redis *cache.Redis
		}{"with Redis", redis})
	}

	for ci, c := range caches {
		for i, tt := range tests {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				st := New(db)
				if c.redis != nil {
					st = st.WithCache(c.redis)
				}
				user := model.User{Email: fmt.Sprintf("stepup-%d-%d@example.com", ci, i), IsActive: true}
				if err := st.CreateUser(ctx, &user); err != nil {
					t.Fatal(err)
				}
				made := model.UserSession{
					TokenHash: newHash(t), UserID: user.ID, AuthenticatedAt: now.Add(-time.Hour),
					Methods: model.StringList{model.AMRPassword}, ExpiresAt: now.Add(time.Hour),
				}
				if err := st.CreateUserSession(ctx, &made); err != nil {
					t.Fatal(err)
				}

				read, err := st.UserSessionByHash(ctx, made.TokenHash)
				if err != nil {
					t.Fatal(err)
				}
				at := tt.end(st, read)
				var before model.UserSession
				if err := db.First(&before, "id = ?", made.ID).Error; err != nil {
					t.Fatal(err)
				}

				fresh := newHash(t)
				ok, err := st.StepUpUserSession(ctx, read, fresh, model.WithSecondFactor(read.Methods, model.AMROneTimeCode), at)
				if err != nil {
					t.Fatal(err)
				}
				if ok {
					t.Fatal("the step-up succeeded on a session that had ended")
				}

				var after model.UserSession
				if err := db.First(&after, "id = ?", made.ID).Error; err != nil {
					t.Fatal(err)
				}
				if after.TokenHash != before.TokenHash || !after.AuthenticatedAt.Equal(before.AuthenticatedAt) ||
					model.ProvedSecondFactor(after.Methods) != model.ProvedSecondFactor(before.Methods) {
					t.Errorf("the row changed: token %t, authenticated_at %v → %v, methods %v → %v",
						after.TokenHash != before.TokenHash, before.AuthenticatedAt, after.AuthenticatedAt, before.Methods, after.Methods)
				}
				if read.TokenHash != made.TokenHash {
					t.Error("the session read was given the new token although nothing was written")
				}
				if _, err := st.UserSessionByHash(ctx, fresh); !errors.Is(err, ErrNotFound) {
					t.Errorf("the new token names a session (%v), want none", err)
				}
			})
		}
	}
}

func newHash(t *testing.T) string {
	t.Helper()
	_, hash, err := model.NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
