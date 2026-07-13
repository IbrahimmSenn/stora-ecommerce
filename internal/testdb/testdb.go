// Package testdb provides a shared connection helper for integration tests.
// Tests that use it are gated behind the `integration` build tag and skip
// unless TEST_DATABASE_URL points at a migrated Postgres.
package testdb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool connects to the integration database named by TEST_DATABASE_URL. If the
// variable is unset the test is skipped, so `go test ./...` stays green on a
// machine without a database. The pool is closed at test cleanup.
func Pool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test db: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
