// publisher.go — confirm-mode AMQP publisher. Publish blocks until the
// broker acks the message, so callers (e.g. the Stripe webhook handler)
// don't return 200 until the event is durable.
package messaging

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher publishes messages on a single confirm-mode channel.
type Publisher struct {
	ch *amqp.Channel
}

// NewPublisher opens a new channel on conn, puts it in confirm mode, and
// returns a Publisher. The caller is responsible for closing the channel
// via Close().
func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open publisher channel: %w", err)
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}
	return &Publisher{ch: ch}, nil
}

// Channel exposes the underlying channel so topology can be declared on it
// at boot before any publishes.
func (p *Publisher) Channel() *amqp.Channel { return p.ch }

// Publish marshals body to the given exchange + routing key and waits for
// the broker confirm. Returns an error if the broker NACKs or the wait
// times out (5s).
func (p *Publisher) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	confirm, err := p.ch.PublishWithDeferredConfirmWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("publish %s/%s: %w", exchange, routingKey, err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	acked, err := confirm.WaitContext(waitCtx)
	if err != nil {
		return fmt.Errorf("wait confirm %s/%s: %w", exchange, routingKey, err)
	}
	if !acked {
		return fmt.Errorf("broker nacked publish to %s/%s", exchange, routingKey)
	}
	return nil
}

func (p *Publisher) Close() error { return p.ch.Close() }
