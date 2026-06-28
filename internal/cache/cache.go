// Package cache provides a small key/value cache abstraction with an in-memory
// default and an optional Redis backing. The interface lets the app stay a
// single binary in development while supporting a shared cache across instances
// once it's scaled horizontally — flip it on by setting REDIS_URL.
package cache

import (
	"context"
	"encoding/json"
	"time"
)

// Cache is a minimal byte-oriented store. Get's second return is false on a
// miss. Implementations must be safe for concurrent use.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// GetJSON reads key and unmarshals it into T. Returns ok=false on a miss or any
// error (callers treat the cache as best-effort and fall back to the source).
func GetJSON[T any](ctx context.Context, c Cache, key string) (T, bool) {
	var zero T
	b, ok, err := c.Get(ctx, key)
	if err != nil || !ok {
		return zero, false
	}
	if err := json.Unmarshal(b, &zero); err != nil {
		return zero, false
	}
	return zero, true
}

// SetJSON marshals v and stores it under key. Errors are returned but callers
// generally ignore them — a failed cache write is not a request failure.
func SetJSON[T any](ctx context.Context, c Cache, key string, v T, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, b, ttl)
}
