// conn.go — AMQP connection helper with bounded boot-time retry. The
// rabbitmq container has a healthcheck in docker-compose, so a brief window
// where dial fails is rare but possible — we retry to absorb it.
package messaging

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connect dials the broker, retrying on failure with a fixed 1s backoff
// until ctx is cancelled or the dial succeeds.
func Connect(ctx context.Context, url string) (*amqp.Connection, error) {
	const backoff = time.Second
	attempt := 0
	for {
		attempt++
		conn, err := amqp.Dial(url)
		if err == nil {
			log.Printf("messaging: connected to rabbitmq (attempt %d)", attempt)
			return conn, nil
		}
		log.Printf("messaging: dial attempt %d failed: %v", attempt, err)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("rabbitmq dial: %w", ctx.Err())
		case <-time.After(backoff):
		}
	}
}
