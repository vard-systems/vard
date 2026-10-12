package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	AdminSession = "admin"
	UserSession  = "user"
)

const SessionTTL = 10 * time.Minute

// Sessions are found by token hash, which no write knows in advance, so they
// are not kept in generations. Instead a reader that missed only fills an empty
// key (SET NX) and a write always overwrites (SET), revoked sessions included.
//
// So a reader that loaded a session just before it was revoked cannot put it
// back.

func (c *Cache) Session(ctx context.Context, kind, hash string, dst any) bool {
	if c == nil || !c.available() || !c.settle(ctx) {
		return false
	}

	raw, err := c.client.Get(ctx, c.sessionKey(kind, hash)).Bytes()
	switch {
	case errors.Is(err, redis.Nil):
		c.recovered()
		return false
	case err != nil:
		c.failed(err)
		return false
	}

	c.recovered()

	return json.Unmarshal(raw, dst) == nil
}

// FillSession keeps a session a reader has just loaded from the database,
// unless a write has put one there meanwhile.
func (c *Cache) FillSession(ctx context.Context, kind, hash string, session any, expires time.Time) {
	if c == nil || !c.available() || !c.settle(ctx) {
		return
	}

	raw, err := json.Marshal(session)
	if err != nil {
		c.log.Error("cache: a session would not encode", "kind", kind, "error", err)
		return
	}

	if err := c.client.SetNX(ctx, c.sessionKey(kind, hash), raw, sessionTTL(expires)).Err(); err != nil {
		c.failed(err)
	}
}

// PutSession replaces a session with what a write just committed. If Redis is
// down the key is removed once it answers, and this process skips Redis until
// then.
func (c *Cache) PutSession(ctx context.Context, kind, hash string, session any, expires time.Time) {
	if c == nil || hash == "" {
		return
	}

	raw, err := json.Marshal(session)
	if err != nil {
		c.log.Error("cache: a session would not encode", "kind", kind, "error", err)
		c.forgetKeys(err, c.sessionKey(kind, hash))
		return
	}

	err = errUnavailable
	if c.available() && c.settle(ctx) {
		err = c.client.Set(ctx, c.sessionKey(kind, hash), raw, sessionTTL(expires)).Err()
	}

	if err != nil {
		c.failed(err)
		c.forgetKeys(err, c.sessionKey(kind, hash))
	}
}

func (c *Cache) forgetKeys(err error, keys ...string) {
	c.mu.Lock()
	if c.pendingKeys == nil {
		c.pendingKeys = map[string]bool{}
	}
	for _, key := range keys {
		c.pendingKeys[key] = true
	}
	c.mu.Unlock()

	c.log.Error("cache: could not update sessions; not using redis until they are removed", "sessions", len(keys), "error", err)
}

func (c *Cache) sessionKey(kind, hash string) string {
	return c.key("session", kind, hash)
}

func sessionTTL(expires time.Time) time.Duration {
	return max(time.Second, min(SessionTTL, time.Until(expires)))
}

func (c *Cache) DropSessions(ctx context.Context, kind string, hashes ...string) {
	if c == nil || len(hashes) == 0 {
		return
	}

	keys := make([]string, len(hashes))
	for i, hash := range hashes {
		keys[i] = c.sessionKey(kind, hash)
	}

	err := errUnavailable
	if c.available() {
		err = c.client.Unlink(ctx, keys...).Err()
	}

	if err != nil {
		c.failed(err)
		c.forgetKeys(err, keys...)
	}
}
