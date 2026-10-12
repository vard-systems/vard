package ratelimit

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"loginer/internal/api/respond"
	"loginer/internal/cache"
)

var RateLimited = respond.Define(http.StatusTooManyRequests, "rate_limited", respond.Both)

type Limiter struct {
	perMinute int
	rate      float64
	burst     float64

	shared *cache.Cache
	scope  string

	mu      sync.Mutex
	buckets map[string]*bucket
	swept   time.Time

	now func() time.Time
}

type bucket struct {
	tokens float64
	seen   time.Time
}

func New(perMinute int) *Limiter {
	return &Limiter{
		perMinute: perMinute,
		rate:      float64(perMinute) / 60,
		burst:     float64(perMinute),
		buckets:   map[string]*bucket{},
		now:       time.Now,
	}
}

func (l *Limiter) Shared(c *cache.Cache, scope string) *Limiter {
	l.shared = c
	l.scope = scope
	return l
}

func (l *Limiter) Allow(ctx context.Context, key string) (bool, time.Duration) {
	if l.burst <= 0 {
		return true, 0
	}

	if l.shared != nil {
		ok, wait, err := l.shared.Take(ctx, l.scope+":"+key, l.perMinute)
		if err == nil {
			return ok, wait
		}
	}

	return l.allowLocally(key)
}

func (l *Limiter) allowLocally(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, seen: now}
		l.buckets[key] = b
	}

	b.tokens = min(l.burst, b.tokens+now.Sub(b.seen).Seconds()*l.rate)
	b.seen = now

	if b.tokens < 1 {
		return false, time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	}

	b.tokens--
	return true, 0
}

// bucketFor groups IPv6 clients by /64, since one client can rotate through a
// whole /64.
func bucketFor(address string) string {
	ip := net.ParseIP(address)
	if ip == nil || ip.To4() != nil {
		return address
	}

	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.swept) < time.Minute {
		return
	}
	l.swept = now

	full := time.Duration(l.burst / l.rate * float64(time.Second))
	for key, b := range l.buckets {
		if now.Sub(b.seen) > full {
			delete(l.buckets, key)
		}
	}
}

// Middleware answers 429 with Retry-After over the limit. It keys on Gin's
// ClientIP, so behind a proxy LOGINER_TRUSTED_PROXIES must name it.
func (l *Limiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := l.Allow(c.Request.Context(), bucketFor(c.ClientIP()))
		if !ok {
			seconds := int(wait/time.Second) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			respond.Abort(c, RateLimited, "seconds", seconds)
			return
		}

		c.Next()
	}
}
