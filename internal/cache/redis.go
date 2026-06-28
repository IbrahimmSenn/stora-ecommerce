package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis backs the cache with a shared Redis server so multiple app instances
// see the same entries. Activated only when REDIS_URL is set.
type Redis struct {
	client *redis.Client
	prefix string
}

// NewRedis wraps a go-redis client. prefix namespaces keys so the cache and any
// other Redis users don't collide.
func NewRedis(client *redis.Client, prefix string) *Redis {
	return &Redis{client: client, prefix: prefix}
}

func (r *Redis) Get(ctx context.Context, key string) ([]byte, bool, error) {
	b, err := r.client.Get(ctx, r.prefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}

func (r *Redis) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return r.client.Set(ctx, r.prefix+key, val, ttl).Err()
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.prefix+key).Err()
}
