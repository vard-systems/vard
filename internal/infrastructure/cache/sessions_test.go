package cache

import (
	"context"
	"testing"
	"time"
)

type testSession struct {
	ID        string     `json:"id"`
	RevokedAt *time.Time `json:"revoked_at"`
}

// A reader that loaded a session just before it was revoked cannot put it
// back, whichever of the two reaches Redis first.
func TestLiveARevokedSessionStaysRevoked(t *testing.T) {
	c := liveRedis(t).Sessions
	ctx := context.Background()
	expires := time.Now().Add(time.Hour)
	revoked := time.Now()

	tests := []struct {
		name        string
		readerFirst bool
	}{
		{name: "the reader's fill lands before the revocation", readerFirst: true},
		{name: "the revocation lands before the reader's fill", readerFirst: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := tt.name

			fill := func() { c.FillSession(ctx, UserSession, hash, testSession{ID: "s"}, expires) }
			revoke := func() {
				c.PutSession(ctx, UserSession, hash, testSession{ID: "s", RevokedAt: &revoked}, expires)
			}

			if tt.readerFirst {
				fill()
				revoke()
			} else {
				revoke()
				fill()
			}

			var got testSession
			if !c.Session(ctx, UserSession, hash, &got) {
				t.Fatal("the session is not in redis at all")
			}
			if got.RevokedAt == nil {
				t.Error("the session reads as active after it was revoked")
			}
		})
	}
}

// A session is kept no longer than it lasts, and the two kinds are apart.
func TestLiveSessionsAreKeptByKind(t *testing.T) {
	c := liveRedis(t).Sessions
	ctx := context.Background()

	c.FillSession(ctx, AdminSession, "abc", testSession{ID: "admin"}, time.Now().Add(time.Minute))

	var got testSession
	if c.Session(ctx, UserSession, "abc", &got) {
		t.Error("an administrator's session was found as a user's")
	}
	if !c.Session(ctx, AdminSession, "abc", &got) || got.ID != "admin" {
		t.Fatalf("Session() = %+v, want the one filled", got)
	}

	ttl := c.client.PTTL(ctx, c.sessionKey(AdminSession, "abc")).Val()
	if ttl <= 0 || ttl > time.Minute {
		t.Errorf("ttl = %v, want at most the minute the session has left", ttl)
	}
}

// A write that could not reach Redis removes the session once it can, before
// anything is read.
func TestLiveAPutThatFailedRemovesTheSessionFirst(t *testing.T) {
	c := liveRedis(t).Sessions
	ctx := context.Background()

	c.FillSession(ctx, UserSession, "abc", testSession{ID: "active"}, time.Now().Add(time.Hour))
	c.pendingKeys = map[string]bool{c.sessionKey(UserSession, "abc"): true}

	var got testSession
	if c.Session(ctx, UserSession, "abc", &got) {
		t.Errorf("Session() = %+v, the session from before a write that failed", got)
	}
	if len(c.pendingKeys) != 0 {
		t.Errorf("pending keys = %v after Redis answered, want them removed", c.pendingKeys)
	}
}
