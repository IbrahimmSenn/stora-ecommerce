// consumer.go — pulls messages off a queue, runs a handler with bounded
// in-process retry, then ACKs or NACKs (to DLX) based on the result.
package messaging

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/tracing"
)

// Handler processes one delivery. Returning nil acks; returning a non-nil
// error triggers in-process retry up to MaxAttempts. After MaxAttempts the
// message is NACK'd without requeue and routed to the queue's DLX.
type Handler func(ctx context.Context, routingKey string, body []byte) error

// Consumer reads from a single queue on its own channel.
type Consumer struct {
	ch    *amqp.Channel
	queue string
	// Backoffs is the wait between attempts. len(Backoffs)+1 == MaxAttempts.
	Backoffs []time.Duration
}

func NewConsumer(conn *amqp.Connection, queue string) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open consumer channel: %w", err)
	}
	// One unacked message at a time per consumer — the work (email send) is
	// fast and ordering doesn't matter, so prefetch=1 keeps the failure
	// blast radius small.
	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("set qos: %w", err)
	}
	return &Consumer{
		ch:       ch,
		queue:    queue,
		Backoffs: []time.Duration{200 * time.Millisecond, time.Second, 5 * time.Second},
	}, nil
}

// Channel exposes the underlying channel for topology declaration at boot.
func (c *Consumer) Channel() *amqp.Channel { return c.ch }

// Run blocks consuming until ctx is cancelled. On cancellation it cancels
// the AMQP consumer, lets in-flight deliveries finish, then returns nil.
func (c *Consumer) Run(ctx context.Context, h Handler) error {
	deliveries, err := c.ch.ConsumeWithContext(ctx, c.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", c.queue, err)
	}
	log.Printf("messaging: consuming %s", c.queue)

	for {
		select {
		case <-ctx.Done():
			log.Printf("messaging: stopping consumer for %s", c.queue)
			return nil
		case d, ok := <-deliveries:
			if !ok {
				// Channel closed by broker or connection drop. Surface this
				// so the supervisor in main.go can decide what to do.
				return fmt.Errorf("delivery channel closed for %s", c.queue)
			}
			c.dispatch(ctx, d, h)
		}
	}
}

func (c *Consumer) dispatch(ctx context.Context, d amqp.Delivery, h Handler) {
	// Join the publisher's trace via the context in the message headers.
	ctx, endSpan := tracing.StartConsumeSpan(ctx, c.queue, d.RoutingKey, d.Headers)
	var spanErr error
	defer func() { endSpan(spanErr) }()

	maxAttempts := len(c.Backoffs) + 1
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := h(ctx, d.RoutingKey, d.Body); err != nil {
			lastErr = err
			log.Printf("messaging: handler attempt %d/%d for %s failed: %v", attempt, maxAttempts, d.RoutingKey, err)
			if attempt < maxAttempts {
				select {
				case <-ctx.Done():
					// Shutdown mid-retry — NACK with requeue so another
					// instance (or this one on restart) can try again.
					_ = d.Nack(false, true)
					spanErr = ctx.Err()
					return
				case <-time.After(c.Backoffs[attempt-1]):
				}
			}
			continue
		}
		if err := d.Ack(false); err != nil {
			log.Printf("messaging: ack failed for %s: %v", d.RoutingKey, err)
		}
		return
	}
	log.Printf("messaging: giving up on %s after %d attempts (last err: %v) — routing to DLX", d.RoutingKey, maxAttempts, lastErr)
	spanErr = lastErr
	if err := d.Nack(false, false); err != nil {
		log.Printf("messaging: nack failed for %s: %v", d.RoutingKey, err)
	}
}

func (c *Consumer) Close() error { return c.ch.Close() }
