package ratelimit

import (
	"context"
	"testing"
	"time"

	"loginer/internal/cache/cachetest"
)

func TestLimiterAllowsABurstThenRefills(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	l := New(3)
	l.now = func() time.Time { return now }

	for i := range 3 {
		if ok, _ := l.Allow(context.Background(), "203.0.113.7"); !ok {
			t.Fatalf("request %d refused inside the burst", i+1)
		}
	}

	ok, wait := l.Allow(context.Background(), "203.0.113.7")
	if ok || wait <= 0 || wait > 20*time.Second {
		t.Fatalf("fourth request = %v, wait %v; want refused, about 20s", ok, wait)
	}

	if ok, _ := l.Allow(context.Background(), "198.51.100.1"); !ok {
		t.Error("another address shares the first one's budget")
	}

	now = now.Add(20 * time.Second)
	if ok, _ := l.Allow(context.Background(), "203.0.113.7"); !ok {
		t.Error("no token after a third of a minute at 3 a minute")
	}
}

func TestLimiterOffAtZero(t *testing.T) {
	l := New(0)
	for range 1000 {
		if ok, _ := l.Allow(context.Background(), "x"); !ok {
			t.Fatal("a zero limit refused a request")
		}
	}
}

func TestLimiterForgetsIdleAddresses(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	l := New(60)
	l.now = func() time.Time { return now }

	l.Allow(context.Background(), "a")
	now = now.Add(2 * time.Minute)
	l.Allow(context.Background(), "b")

	if _, kept := l.buckets["a"]; kept {
		t.Error("an address idle long enough to refill was kept")
	}
}

// Two server processes with a Redis between them share one budget per
// address: what one spent, the other cannot spend again.
func TestLiveLimitIsSharedBetweenProcesses(t *testing.T) {
	c := cachetest.Require(t).Sessions

	first := New(2).Shared(c, "public")
	second := New(2).Shared(c, "public")
	ctx := context.Background()

	if ok, _ := first.Allow(ctx, "203.0.113.7"); !ok {
		t.Fatal("the first attempt was refused")
	}
	if ok, _ := second.Allow(ctx, "203.0.113.7"); !ok {
		t.Fatal("the second attempt, at the other process, was refused")
	}
	if ok, _ := first.Allow(ctx, "203.0.113.7"); ok {
		t.Error("a third attempt was allowed: each process kept its own count")
	}
}

// A Redis that stops answering leaves each process counting on its own —
// a weaker limit, never none.
func TestLiveLimitFallsBackWithoutRedis(t *testing.T) {
	c := cachetest.Require(t).Sessions
	l := New(1).Shared(c, "public")
	ctx := context.Background()

	_ = c.Close()

	if ok, _ := l.Allow(ctx, "203.0.113.7"); !ok {
		t.Fatal("the first attempt was refused with Redis gone")
	}
	if ok, _ := l.Allow(ctx, "203.0.113.7"); ok {
		t.Error("the limit stopped limiting when Redis went away")
	}
}
