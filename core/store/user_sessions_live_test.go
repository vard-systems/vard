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
// brought back under a new token, with Redis configured or not.
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

// A session is found by its token hash through a unique index, on every
// request, since none is cached; and the families a session's end reaches
// are found by an index too.
func TestLiveSessionsAreLookedUpByIndex(t *testing.T) {
	db, _ := databasetest.Open(t)

	for _, index := range []struct{ table, definition string }{
		{"user_sessions", "CREATE UNIQUE INDEX idx_user_sessions_token_hash ON public.user_sessions USING btree (token_hash)"},
		{"admin_sessions", "CREATE UNIQUE INDEX idx_admin_sessions_token_hash ON public.admin_sessions USING btree (token_hash)"},
		{"refresh_tokens", "CREATE INDEX idx_refresh_tokens_session_id ON public.refresh_tokens USING btree (session_id)"},
	} {
		var found int64
		if err := db.Raw("SELECT count(*) FROM pg_indexes WHERE tablename = ? AND indexdef = ?", index.table, index.definition).Scan(&found).Error; err != nil {
			t.Fatal(err)
		}
		if found != 1 {
			t.Errorf("%s has no index: %s", index.table, index.definition)
		}
	}
}

// An administrator's session records being used at most once a SeenEvery,
// so a busy panel does not write a row on every request, and it is written
// to the database, which every server reads.
func TestLiveLastSeenIsWrittenAtMostOnceASeenEvery(t *testing.T) {
	db, _ := databasetest.Open(t)
	st := New(db)
	ctx := context.Background()
	start := time.Now().UTC().Truncate(time.Microsecond)

	admin := model.Admin{Username: "seen@example.com", Email: "seen@example.com", FirstName: "Seen", PasswordHash: "x", Status: model.StatusActive}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	session := model.AdminSession{AdminID: admin.ID, TokenHash: newHash(t), ExpiresAt: start.Add(time.Hour), LastSeenAt: start}
	if err := st.CreateAdminSession(ctx, &session); err != nil {
		t.Fatal(err)
	}

	stored := func() time.Time {
		t.Helper()
		read, err := st.AdminSessionByTokenHash(ctx, session.TokenHash)
		if err != nil {
			t.Fatal(err)
		}
		return read.LastSeenAt.UTC()
	}

	for _, step := range []struct {
		name  string
		at    time.Time
		wants time.Time
	}{
		{"used again within a SeenEvery", start.Add(model.SeenEvery / 2), start},
		{"used just past a SeenEvery", start.Add(model.SeenEvery + time.Second), start.Add(model.SeenEvery + time.Second)},
		{"used again straight after", start.Add(model.SeenEvery + 2*time.Second), start.Add(model.SeenEvery + time.Second)},
	} {
		read, err := st.AdminSessionByTokenHash(ctx, session.TokenHash)
		if err != nil {
			t.Fatal(err)
		}
		if err := st.TouchAdminSession(ctx, read, step.at); err != nil {
			t.Fatal(err)
		}
		if got := stored(); !got.Equal(step.wants) {
			t.Errorf("%s: last_seen_at = %v, want %v", step.name, got, step.wants)
		}
	}
}
