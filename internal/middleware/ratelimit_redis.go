package middleware

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// tokenBucketScript is an atomic token-bucket refill+consume. It stores the
// current token count and last-refill timestamp per key, refills based on
// elapsed time, and consumes one token if available. Returns 1 (allowed) or 0.
var tokenBucketScript = redis.NewScript(`
local rate     = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now      = tonumber(ARGV[3])
local data = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])
if tokens == nil then
  tokens = capacity
  ts = now
end
local delta = now - ts
if delta < 0 then delta = 0 end
tokens = math.min(capacity, tokens + delta * rate)
local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end
redis.call('HSET', KEYS[1], 'tokens', tokens, 'ts', now)
local ttl = 3600
if rate > 0 then ttl = math.ceil(capacity / rate) + 10 end
redis.call('EXPIRE', KEYS[1], ttl)
return allowed
`)

// redisStore enforces one shared token bucket per client across all app
// instances. prefix namespaces keys so the two limiters (general/auth) and the
// cache don't collide in the same Redis.
type redisStore struct {
	client *redis.Client
	prefix string
	rps    float64
	burst  int
}

func newRedisStore(client *redis.Client, prefix string, rps float64, burst int) *redisStore {
	return &redisStore{client: client, prefix: prefix, rps: rps, burst: burst}
}

func (s *redisStore) allow(ctx context.Context, key string) bool {
	now := float64(time.Now().UnixNano()) / 1e9
	res, err := tokenBucketScript.Run(ctx, s.client,
		[]string{s.prefix + key},
		s.rps, float64(s.burst), now,
	).Int()
	if err != nil {
		// Fail open: a Redis hiccup must not take the site down. Log so the
		// outage is visible.
		log.Printf("ratelimit: redis error, allowing request: %v", err)
		return true
	}
	return res == 1
}

// NewRedisRateLimiter builds a limiter whose state lives in Redis, shared across
// instances. Used when REDIS_URL is configured.
func NewRedisRateLimiter(client *redis.Client, prefix string, rps float64, burst int) *RateLimiter {
	return &RateLimiter{store: newRedisStore(client, prefix, rps, burst)}
}
