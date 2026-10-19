package cache

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// What a super admin can see of Redis and do to it, always under this server's
// prefix. A generation counter is never removed (generation zero would come
// back), and nothing in the session database is written by hand, since that
// would sign somebody in. Removing a key is always safe.

const (
	KindEntry      = "entry"
	KindStale      = "stale"
	KindGeneration = "generation"
	KindSession    = "session"
	KindRateLimit  = "ratelimit"
	KindOther      = "other"
)

var (
	ErrNoRedis       = errors.New("redis is not configured")
	ErrNoKey         = errors.New("no such key")
	ErrGenerationKey = errors.New("a generation counter cannot be removed; clear its group instead")
	ErrNotEditable   = errors.New("only a cached value in the cache database can be edited")
)

var Groups = map[string][]string{
	CacheDatabase:   {Languages, Organization, LoginFlows, SocialButtons, SSOButtons, OTPSettings},
	SessionDatabase: {Admins, Clients, AdminSecurity, Grants},
}

type Key struct {
	Name     string
	Kind     string
	Group    string
	Entry    string
	TTL      time.Duration
	Size     int64
	Editable bool
}

type Value struct {
	Key
	Type  string
	Value json.RawMessage
}

type GroupStats struct {
	Name       string
	Generation int64
	Entries    int64
	Stale      int64
}

type Stats struct {
	Name      string
	Number    int
	Available bool
	Keys      int64
	Kinds     map[string]int64
	Groups    []GroupStats
	Sessions  map[string]int64
	Memory    int64
	Version   string
}

func (c *Cache) Stats(ctx context.Context) (Stats, error) {
	if c == nil {
		return Stats{}, ErrNoRedis
	}

	stats := Stats{Name: c.name, Number: c.number, Kinds: map[string]int64{}, Sessions: map[string]int64{}}

	generations, err := c.generations(ctx, Groups[c.name])
	if err != nil {
		return stats, err
	}
	stats.Available = true

	groups := map[string]*GroupStats{}
	for _, name := range Groups[c.name] {
		groups[name] = &GroupStats{Name: name, Generation: generations[name]}
	}

	iter := c.client.Scan(ctx, 0, c.prefix+"*", 1000).Iterator()
	for iter.Next(ctx) {
		key := c.describe(strings.TrimPrefix(iter.Val(), c.prefix), generations)
		stats.Keys++
		stats.Kinds[key.Kind]++

		switch key.Kind {
		case KindEntry, KindStale:
			group, ok := groups[key.Group]
			if !ok {
				group = &GroupStats{Name: key.Group}
				groups[key.Group] = group
			}
			if key.Kind == KindEntry {
				group.Entries++
			} else {
				group.Stale++
			}
		case KindSession:
			stats.Sessions[key.Group]++
		}
	}
	if err := iter.Err(); err != nil {
		return stats, err
	}

	for _, name := range Groups[c.name] {
		stats.Groups = append(stats.Groups, *groups[name])
		delete(groups, name)
	}
	for _, group := range groups {
		stats.Groups = append(stats.Groups, *group)
	}

	stats.Memory, stats.Version = c.server(ctx)

	return stats, nil
}

type KeyQuery struct {
	Kind   string
	Group  string
	Search string
	Cursor uint64
	Limit  int
}

func (c *Cache) Keys(ctx context.Context, q KeyQuery) ([]Key, uint64, error) {
	if c == nil {
		return nil, 0, ErrNoRedis
	}

	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	generations, err := c.generations(ctx, Groups[c.name])
	if err != nil {
		return nil, 0, err
	}

	pattern := c.prefix + keyPattern(q.Kind, q.Group)

	var (
		keys   []Key
		cursor = q.Cursor
	)
	// A bounded number of steps: a pattern that matches little in a large
	// database would otherwise walk all of it for one page.
	for range 50 {
		names, next, err := c.client.Scan(ctx, cursor, pattern, 500).Result()
		if err != nil {
			return nil, 0, err
		}

		for _, name := range names {
			key := c.describe(strings.TrimPrefix(name, c.prefix), generations)
			if q.Kind != "" && key.Kind != q.Kind && !(q.Kind == KindEntry && key.Kind == KindStale) {
				continue
			}
			if q.Search != "" && !strings.Contains(strings.ToLower(key.Name), strings.ToLower(q.Search)) {
				continue
			}
			keys = append(keys, key)
		}

		cursor = next
		if cursor == 0 || len(keys) >= limit {
			break
		}
	}

	if err := c.measure(ctx, keys); err != nil {
		return nil, 0, err
	}

	return keys, cursor, nil
}

func (c *Cache) Read(ctx context.Context, name string) (Value, error) {
	if c == nil {
		return Value{}, ErrNoRedis
	}

	generations, err := c.generations(ctx, Groups[c.name])
	if err != nil {
		return Value{}, err
	}

	keys := []Key{c.describe(name, generations)}
	if err := c.measure(ctx, keys); err != nil {
		return Value{}, err
	}

	full := c.prefix + name
	kind, err := c.client.Type(ctx, full).Result()
	if err != nil {
		return Value{}, err
	}

	value := Value{Key: keys[0], Type: kind}

	switch kind {
	case "none":
		return Value{}, ErrNoKey
	case "string":
		raw, err := c.client.Get(ctx, full).Bytes()
		if err != nil {
			return Value{}, err
		}
		if !json.Valid(raw) {
			raw, _ = json.Marshal(string(raw))
		}
		value.Value = raw
	case "hash":
		fields, err := c.client.HGetAll(ctx, full).Result()
		if err != nil {
			return Value{}, err
		}
		value.Value, _ = json.Marshal(fields)
	default:
		value.Value, _ = json.Marshal("a " + kind + " this server does not write")
	}

	return value, nil
}

func (c *Cache) Write(ctx context.Context, name string, value json.RawMessage, ttl time.Duration) error {
	if c == nil {
		return ErrNoRedis
	}

	generations, err := c.generations(ctx, Groups[c.name])
	if err != nil {
		return err
	}

	if key := c.describe(name, generations); !key.Editable {
		return ErrNotEditable
	}

	full := c.prefix + name
	exists, err := c.client.Exists(ctx, full).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrNoKey
	}

	if ttl > 0 {
		return c.client.Set(ctx, full, []byte(value), ttl).Err()
	}

	return c.client.SetArgs(ctx, full, []byte(value), redis.SetArgs{KeepTTL: true}).Err()
}

func (c *Cache) Delete(ctx context.Context, name string) (bool, error) {
	if c == nil {
		return false, ErrNoRedis
	}

	if c.describe(name, nil).Kind == KindGeneration {
		return false, ErrGenerationKey
	}

	removed, err := c.client.Unlink(ctx, c.prefix+name).Result()

	return removed > 0, err
}

func (c *Cache) Clear(ctx context.Context, group string) error {
	if c == nil {
		return ErrNoRedis
	}

	return c.incr(ctx, []string{group})
}

func (c *Cache) Flush(ctx context.Context) error {
	if c == nil {
		return nil
	}

	iter := c.client.Scan(ctx, 0, c.prefix+"*", 500).Iterator()

	var batch []string
	for iter.Next(ctx) {
		batch = append(batch, iter.Val())

		if len(batch) == 500 {
			if err := c.client.Unlink(ctx, batch...).Err(); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}

	if len(batch) > 0 {
		return c.client.Unlink(ctx, batch...).Err()
	}

	return nil
}

func (c *Cache) describe(name string, generations map[string]int64) Key {
	key := Key{Name: name, Kind: KindOther, TTL: -1}

	parts := strings.SplitN(name, ":", 4)
	switch {
	case len(parts) == 3 && parts[0] == "cache" && parts[2] == "generation":
		key.Kind, key.Group = KindGeneration, parts[1]
	case len(parts) == 4 && parts[0] == "cache" && strings.HasPrefix(parts[2], "v"):
		key.Group, key.Entry = parts[1], parts[3]
		generation, _ := strconv.ParseInt(strings.TrimPrefix(parts[2], "v"), 10, 64)
		if generation == generations[key.Group] {
			key.Kind = KindEntry
			key.Editable = c.name == CacheDatabase
		} else {
			key.Kind = KindStale
		}
	case len(parts) >= 3 && parts[0] == "session":
		key.Kind, key.Group, key.Entry = KindSession, parts[1], strings.Join(parts[2:], ":")
	case len(parts) >= 3 && parts[0] == "ratelimit":
		key.Kind, key.Group, key.Entry = KindRateLimit, parts[1], strings.Join(parts[2:], ":")
	}

	return key
}

func (c *Cache) measure(ctx context.Context, keys []Key) error {
	if len(keys) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	ttls := make([]*redis.DurationCmd, len(keys))
	sizes := make([]*redis.IntCmd, len(keys))
	for i, key := range keys {
		ttls[i] = pipe.PTTL(ctx, c.prefix+key.Name)
		sizes[i] = pipe.MemoryUsage(ctx, c.prefix+key.Name)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) && !isCommandError(err) {
		return err
	}

	for i := range keys {
		if ttl, err := ttls[i].Result(); err == nil && ttl >= 0 {
			keys[i].TTL = ttl
		}
		keys[i].Size, _ = sizes[i].Result()
	}

	return nil
}

func (c *Cache) generations(ctx context.Context, groups []string) (map[string]int64, error) {
	current := make(map[string]int64, len(groups))
	if len(groups) == 0 {
		return current, nil
	}

	names := make([]string, len(groups))
	for i, group := range groups {
		names[i] = c.generationKey(group)
	}

	values, err := c.client.MGet(ctx, names...).Result()
	if err != nil {
		return nil, err
	}

	for i, value := range values {
		if text, ok := value.(string); ok {
			current[groups[i]], _ = strconv.ParseInt(text, 10, 64)
		}
	}

	return current, nil
}

func (c *Cache) server(ctx context.Context) (memory int64, version string) {
	info, err := c.client.Info(ctx, "server", "memory").Result()
	if err != nil {
		return 0, ""
	}

	lines := bufio.NewScanner(strings.NewReader(info))
	for lines.Scan() {
		name, value, ok := strings.Cut(strings.TrimSpace(lines.Text()), ":")
		if !ok {
			continue
		}
		switch name {
		case "used_memory":
			memory, _ = strconv.ParseInt(value, 10, 64)
		case "redis_version":
			version = value
		}
	}

	return memory, version
}

func keyPattern(kind, group string) string {
	switch kind {
	case KindEntry, KindStale:
		if group != "" {
			return "cache:" + escapeGlob(group) + ":v*"
		}
		return "cache:*:v*"
	case KindGeneration:
		return "cache:*:generation"
	case KindSession:
		if group != "" {
			return "session:" + escapeGlob(group) + ":*"
		}
		return "session:*"
	case KindRateLimit:
		if group != "" {
			return "ratelimit:" + escapeGlob(group) + ":*"
		}
		return "ratelimit:*"
	}

	if group != "" {
		return "cache:" + escapeGlob(group) + ":*"
	}

	return "*"
}

func escapeGlob(text string) string {
	return strings.NewReplacer(`\`, `\\`, `*`, `\*`, `?`, `\?`, `[`, `\[`, `]`, `\]`).Replace(text)
}

func isCommandError(err error) bool {
	var redisErr redis.Error
	return errors.As(err, &redisErr)
}
