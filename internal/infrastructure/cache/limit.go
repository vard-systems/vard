package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// takeToken is a token bucket in Redis, taken from atomically so two processes
// cannot spend the last token. It uses Redis's clock, so servers whose clocks
// disagree still agree on the bucket. It answers whether a token was taken and,
// if not, the milliseconds until one is.
var takeToken = redis.NewScript(`
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local time = redis.call('TIME')
local now = tonumber(time[1]) * 1000 + math.floor(tonumber(time[2]) / 1000)

local bucket = redis.call('HMGET', KEYS[1], 'tokens', 'at')
local tokens = tonumber(bucket[1]) or burst
local at = tonumber(bucket[2]) or now

tokens = math.min(burst, tokens + math.max(0, now - at) / 1000 * rate)

local allowed, wait = 0, 0
if tokens >= 1 then
	tokens = tokens - 1
	allowed = 1
else
	wait = math.ceil((1 - tokens) / rate * 1000)
end

redis.call('HSET', KEYS[1], 'tokens', tostring(tokens), 'at', now)
redis.call('PEXPIRE', KEYS[1], math.ceil(burst / rate * 1000))

return {allowed, wait}
`)

func (c *Cache) Take(ctx context.Context, bucket string, perMinute int) (bool, time.Duration, error) {
	if !c.available() {
		return false, 0, errUnavailable
	}

	rate := float64(perMinute) / 60

	result, err := takeToken.Run(ctx, c.client, []string{c.key("ratelimit", bucket)}, rate, perMinute).Int64Slice()
	if err != nil {
		c.failed(err)
		return false, 0, err
	}

	c.recovered()

	return result[0] == 1, time.Duration(result[1]) * time.Millisecond, nil
}
