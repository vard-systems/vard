// Package cachetest opens a throwaway Redis for tests, under a key prefix of its
// own that is flushed afterwards.
package cachetest

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

	"github.com/redis/go-redis/v9"

	"vard/internal/brand"
	"vard/internal/config"
	"vard/internal/infrastructure/cache"
)

const Env = "VARD_TEST_REDIS"

func Open(t *testing.T) *cache.Redis {
	t.Helper()

	addr := os.Getenv(Env)
	if addr == "" {
		return nil
	}

	host, port, _ := strings.Cut(addr, ":")
	number, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("%s = %q, want host:port", Env, addr)
	}

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}

	r, err := cache.Open(context.Background(), config.Redis{
		Host:      host,
		Port:      number,
		CacheDB:   14,
		SessionDB: 15,
		Username:  os.Getenv("VARD_REDIS_USERNAME"),
		Password:  os.Getenv("VARD_REDIS_PASSWORD"),
		Prefix:    brand.Slug + "_test_" + hex.EncodeToString(suffix) + ":",
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		for _, c := range r.Databases() {
			if err := c.Flush(context.Background()); err != nil && !errors.Is(err, redis.ErrClosed) {
				t.Errorf("remove the test's keys: %v", err)
			}
		}
		_ = r.Close()
	})

	return r
}

func Require(t *testing.T) *cache.Redis {
	t.Helper()

	r := Open(t)
	if r == nil {
		t.Skipf("%s is not set", Env)
	}

	return r
}
