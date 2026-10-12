package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// TTL only bounds how long an abandoned generation occupies memory; writes
// forget what they change.
const TTL = time.Hour

func (c *Cache) Get(ctx context.Context, group, entry string, dst any) bool {
	if c == nil || !c.available() || !c.settle(ctx) {
		return false
	}

	key, err := c.entryKey(ctx, group, entry)
	if err != nil {
		c.failed(err)
		return false
	}

	raw, err := c.client.Get(ctx, key).Bytes()
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

func (c *Cache) Set(ctx context.Context, group, entry string, value any) {
	if c == nil || !c.available() || !c.settle(ctx) {
		return
	}

	raw, err := json.Marshal(value)
	if err != nil {
		c.log.Error("cache: a value would not encode", "group", group, "entry", entry, "error", err)
		return
	}

	key, err := c.entryKey(ctx, group, entry)
	if err != nil {
		c.failed(err)
		return
	}

	if err := c.client.Set(ctx, key, raw, TTL).Err(); err != nil {
		c.failed(err)
	}
}

// Forget drops everything cached in the given groups, for every process. It is
// called after a write commits; when Redis is unreachable the groups stay
// pending until it answers.
func (c *Cache) Forget(ctx context.Context, groups ...string) {
	if c == nil || len(groups) == 0 {
		return
	}

	err := errUnavailable
	if c.available() {
		err = c.incr(ctx, groups)
	}

	if err != nil {
		c.failed(err)

		c.mu.Lock()
		if c.pending == nil {
			c.pending = map[string]bool{}
		}
		for _, group := range groups {
			c.pending[group] = true
		}
		c.mu.Unlock()

		c.log.Error("cache: could not forget a group; not using the cache until it is forgotten",
			"groups", groups, "error", err)
	}
}

func (c *Cache) settle(ctx context.Context) bool {
	c.mu.Lock()
	groups := make([]string, 0, len(c.pending))
	for group := range c.pending {
		groups = append(groups, group)
	}
	keys := make([]string, 0, len(c.pendingKeys))
	for key := range c.pendingKeys {
		keys = append(keys, key)
	}
	c.mu.Unlock()

	if len(groups) == 0 && len(keys) == 0 {
		return true
	}

	if len(groups) > 0 {
		if err := c.incr(ctx, groups); err != nil {
			c.failed(err)
			return false
		}
	}
	if len(keys) > 0 {
		if err := c.client.Unlink(ctx, keys...).Err(); err != nil {
			c.failed(err)
			return false
		}
	}

	c.mu.Lock()
	for _, group := range groups {
		delete(c.pending, group)
	}
	for _, key := range keys {
		delete(c.pendingKeys, key)
	}
	c.mu.Unlock()

	c.log.Info("cache: forgot what a write changed while redis was away", "groups", groups, "keys", len(keys))

	return true
}

func (c *Cache) incr(ctx context.Context, groups []string) error {
	pipe := c.client.Pipeline()
	for _, group := range groups {
		pipe.Incr(ctx, c.generationKey(group))
	}

	_, err := pipe.Exec(ctx)

	return err
}

// generationKey never expires, so memory pressure evicts values but never the
// counter that says which may be read.
func (c *Cache) generationKey(group string) string {
	return c.key("cache", group, "generation")
}

func (c *Cache) generation(ctx context.Context, group string) (int64, error) {
	generation, err := c.client.Get(ctx, c.generationKey(group)).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, err
	}

	return generation, nil
}

func (c *Cache) keyAt(group string, generation int64, entry string) string {
	return c.key("cache", group, "v"+strconv.FormatInt(generation, 10), entry)
}

func (c *Cache) entryKey(ctx context.Context, group, entry string) (string, error) {
	generation, err := c.generation(ctx, group)
	if err != nil {
		return "", err
	}

	return c.keyAt(group, generation, entry), nil
}

func (c *Cache) GetAt(ctx context.Context, group, entry string, dst any) (int64, bool) {
	if c == nil || !c.available() || !c.settle(ctx) {
		return 0, false
	}

	generation, err := c.generation(ctx, group)
	if err != nil {
		c.failed(err)
		return 0, false
	}

	raw, err := c.client.Get(ctx, c.keyAt(group, generation, entry)).Bytes()
	switch {
	case errors.Is(err, redis.Nil):
		c.recovered()
		return generation, false
	case err != nil:
		c.failed(err)
		return 0, false
	}

	c.recovered()

	return generation, json.Unmarshal(raw, dst) == nil
}

// SetAt caches an entry in the generation GetAt read. If a write moved the
// group on meanwhile, the value lands where nobody reads it.
func (c *Cache) SetAt(ctx context.Context, group, entry string, value any, generation int64) {
	if c == nil || !c.available() || !c.settle(ctx) {
		return
	}

	raw, err := json.Marshal(value)
	if err != nil {
		c.log.Error("cache: a value would not encode", "group", group, "entry", entry, "error", err)
		return
	}

	if err := c.client.Set(ctx, c.keyAt(group, generation, entry), raw, TTL).Err(); err != nil {
		c.failed(err)
	}
}
