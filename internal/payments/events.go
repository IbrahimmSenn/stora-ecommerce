// events.go — payment event payloads + the EventPublisher port. Concrete
// implementations live elsewhere (AmqpPublisher in amqp_publisher.go); the
// service depends only on this interface so tests can stub it without AMQP.
package payments

import (
	"context"

	"github.com/google/uuid"
)

// PaymentSucceededEvent is published when a Stripe PaymentIntent succeeds.
type PaymentSucceededEvent struct {
	OrderID         uuid.UUID `json:"order_id"`
	PaymentIntentID string    `json:"payment_intent_id"`
	AmountCents     int64     `json:"amount_cents"`
	Currency        string    `json:"currency"`
}

// PaymentFailedEvent is published when a PaymentIntent fails. Code and
// Message come from Stripe's last_payment_error and may be empty.
type PaymentFailedEvent struct {
	OrderID         uuid.UUID `json:"order_id"`
	PaymentIntentID string    `json:"payment_intent_id"`
	AmountCents     int64     `json:"amount_cents"`
	Currency        string    `json:"currency"`
	FailureCode     string    `json:"failure_code,omitempty"`
	FailureMessage  string    `json:"failure_message,omitempty"`
}

// EventPublisher is the slice of the broker the service uses. Returning an
// error blocks the webhook from acking Stripe, so Stripe will retry.
type EventPublisher interface {
	PublishSucceeded(ctx context.Context, evt PaymentSucceededEvent) error
	PublishFailed(ctx context.Context, evt PaymentFailedEvent) error
}
