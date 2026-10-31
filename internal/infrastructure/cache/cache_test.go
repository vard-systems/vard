package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"vard/internal/brand"
	"vard/internal/config"
)

// The tests that need a Redis name it in VARD_TEST_REDIS, as host:port,
// and skip without it, the way the database tests do with
// VARD_TEST_DB_DSN. Each runs under a prefix of its own and removes its
// keys afterwards, so they can share a Redis with a running server.
const testRedisEnv = "VARD_TEST_REDIS"

func liveRedis(t *testing.T) *Redis {
	t.Helper()

	addr := os.Getenv(testRedisEnv)
	if addr == "" {
		t.Skipf("%s is not set", testRedisEnv)
	}

	host, portText, _ := strings.Cut(addr, ":")
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("%s = %q, want host:port", testRedisEnv, addr)
	}

	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix)
	prefix := brand.Slug + "_test_" + hex.EncodeToString(suffix) + ":"

	r, err := Open(context.Background(), config.Redis{Host: host, Port: port, CacheDB: 14, SessionDB: 15, Prefix: prefix},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		for _, c := range r.Databases() {
			if err := c.Flush(ctx); err != nil {
				t.Error(err)
			}
			if left, _ := c.client.Keys(ctx, prefix+"*").Result(); len(left) > 0 {
				t.Errorf("Flush left %v in the %s database", left, c.name)
			}
		}
		_ = r.Close()
	})

	return r
}

// liveCache is the cache database of a live Redis.
func liveCache(t *testing.T) *Cache {
	t.Helper()
	return liveRedis(t).Cache
}

// No Redis configured is a cache that never holds anything, and every method
// is safe to call on it — which is what lets the store use one without
// asking whether there is one.
func TestNoRedisIsAnEmptyCache(t *testing.T) {
	r, err := Open(context.Background(), config.Redis{}, slog.Default())
	if err != nil || r != nil {
		t.Fatalf("Open(no host) = %v, %v; want nil, nil", r, err)
	}
	c := r.CacheDB()

	ctx := context.Background()
	c.Set(ctx, Languages, "all", []string{"en"})
	c.Forget(ctx, Languages)

	var out []string
	if c.Get(ctx, Languages, "all", &out) {
		t.Error("a nil cache had something in it")
	}
	if r.SessionDB() != nil || r.Databases() != nil {
		t.Error("no Redis had a session database")
	}
	if err := r.Close(); err != nil {
		t.Error(err)
	}
}

func TestOpenRefusesARedisThatDoesNotAnswer(t *testing.T) {
	// Nothing listens on port 1.
	_, err := Open(context.Background(), config.Redis{Host: "127.0.0.1", Port: 1}, slog.Default())
	if err == nil {
		t.Fatal("Open() = nothing, want an error naming the address")
	}
	if !strings.Contains(err.Error(), "127.0.0.1:1") {
		t.Errorf("error = %q, want it to say where it looked", err)
	}
}

func TestLiveCacheForgetsAGroupAtOnce(t *testing.T) {
	c := liveCache(t)
	ctx := context.Background()

	type text struct{ Messages map[string]string }

	c.Set(ctx, Languages, "text:ru:id", text{Messages: map[string]string{"action.sign_in": "Войти"}})
	c.Set(ctx, Organization, "settings", brand.Name)

	var got text
	if !c.Get(ctx, Languages, "text:ru:id", &got) || got.Messages["action.sign_in"] != "Войти" {
		t.Fatalf("Get() = %+v, want what was set", got)
	}

	if c.Get(ctx, Languages, "text:de:id", &got) {
		t.Error("a field nobody set was found")
	}

	c.Forget(ctx, Languages)

	if c.Get(ctx, Languages, "text:ru:id", &got) {
		t.Error("a forgotten group still answered")
	}

	var name string
	if !c.Get(ctx, Organization, "settings", &name) || name != brand.Name {
		t.Error("forgetting one group forgot another")
	}

	// A reader that loaded before the write and stores after it writes into
	// the generation that was forgotten, where nobody looks — not over the
	// write.
	stale, err := c.entryKey(ctx, Languages, "all")
	if err != nil {
		t.Fatal(err)
	}
	c.Forget(ctx, Languages)
	c.client.Set(ctx, stale, `["stale"]`, TTL)

	var all []string
	if c.Get(ctx, Languages, "all", &all) {
		t.Errorf("a value written to a forgotten generation was read back: %v", all)
	}
}

func TestLiveTakeSharesOneBucket(t *testing.T) {
	c := liveRedis(t).Sessions
	ctx := context.Background()

	const perMinute = 3

	for i := range perMinute {
		ok, _, err := c.Take(ctx, "public:203.0.113.7", perMinute)
		if err != nil || !ok {
			t.Fatalf("attempt %d = %v, %v; want allowed", i+1, ok, err)
		}
	}

	ok, wait, err := c.Take(ctx, "public:203.0.113.7", perMinute)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("the attempt after the burst was allowed")
	}
	if wait <= 0 || wait > 21e9 {
		t.Errorf("wait = %v, want about the 20 seconds one token takes to refill", wait)
	}

	// Another address, and another server's bucket, are their own.
	if ok, _, _ := c.Take(ctx, "public:203.0.113.8", perMinute); !ok {
		t.Error("another address was refused")
	}
	if ok, _, _ := c.Take(ctx, "admin:203.0.113.7", perMinute); !ok {
		t.Error("the admin server's bucket was spent by the public server")
	}
}

// A forget that could not reach Redis is not lost: the next call makes it
// before it reads anything, so the value from before the write is not served.
func TestLiveAForgetThatFailedIsMadeFirst(t *testing.T) {
	c := liveCache(t)
	ctx := context.Background()

	c.Set(ctx, Languages, "text:ru:id", "before the write")

	// What Forget leaves behind when Redis did not answer it.
	c.pending = map[string]bool{Languages: true}

	var got string
	if c.Get(ctx, Languages, "text:ru:id", &got) {
		t.Errorf("Get() = %q, the value from before a write whose forget was pending", got)
	}
	if len(c.pending) != 0 {
		t.Errorf("pending = %v after Redis answered, want it made", c.pending)
	}

	c.Set(ctx, Languages, "text:ru:id", "after the write")
	if !c.Get(ctx, Languages, "text:ru:id", &got) || got != "after the write" {
		t.Errorf("Get() = %q, want the cache working again", got)
	}
}

// After Redis fails to answer, the cache leaves it alone for the cooldown:
// reads miss and the rate limit is told to count on its own, at once, rather
// than every request of an outage waiting on a dial that will fail.
func TestLiveCooldownAfterAFailure(t *testing.T) {
	c := liveCache(t)
	ctx := context.Background()

	c.Set(ctx, Languages, "all", []string{"en"})
	c.failed(errors.New("redis did not answer"))

	var all []string
	if c.Get(ctx, Languages, "all", &all) {
		t.Error("the cache was read during its cooldown")
	}
	if _, _, err := c.Take(ctx, "public:203.0.113.7", 5); !errors.Is(err, errUnavailable) {
		t.Errorf("Take() during the cooldown = %v, want errUnavailable", err)
	}

	// A write during the cooldown is remembered, and made first once it ends.
	c.Forget(ctx, Languages)

	c.mu.Lock()
	c.downUntil = time.Time{}
	c.mu.Unlock()

	if c.Get(ctx, Languages, "all", &all) {
		t.Errorf("Get() = %v, the value from before a write made during the cooldown", all)
	}
	c.Set(ctx, Languages, "all", []string{"en", "ru"})
	if !c.Get(ctx, Languages, "all", &all) || len(all) != 2 {
		t.Errorf("Get() = %v after the cooldown, want the cache working again", all)
	}
}
