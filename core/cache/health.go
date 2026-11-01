package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// cooldown is how long Redis is skipped after a failure, so an outage costs one
// probe per cooldown instead of a failed dial per request.
const cooldown = 5 * time.Second

var errUnavailable = errors.New("redis is not answering")

func (c *Cache) available() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return time.Now().After(c.downUntil)
}

func (c *Cache) failed(err error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.downUntil = time.Now().Add(cooldown)

	if !c.warned {
		c.log.Warn("cache: redis did not answer; reading the database until it does", "error", err)
		c.warned = true
	}
}

func (c *Cache) recovered() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.warned {
		c.log.Info("cache: redis is answering again")
		c.warned = false
	}
}

type quiet struct{ log *slog.Logger }

func (q quiet) Printf(ctx context.Context, format string, v ...any) {
	q.log.DebugContext(ctx, "redis: "+fmt.Sprintf(format, v...))
}
