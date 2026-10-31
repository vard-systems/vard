package ratelimit

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
)

// The limiter sits in front of every sign-in. Without Redis it is one mutex
// per process, so this measures it under every core at once, spread over
// many addresses the way real traffic is.
func BenchmarkAllowInMemory(b *testing.B) {
	l := New(1_000_000)
	ctx := context.Background()
	var next atomic.Uint64

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Allow(ctx, "198.51.100."+strconv.FormatUint(next.Add(1)%250, 10))
		}
	})
}
