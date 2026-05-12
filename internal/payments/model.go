package payments

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending   = "pending"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
	StatusRefunded  = "refunded"
)

type Payment struct {
	ID                    uuid.UUID  `json:"id"`
	OrderID               uuid.UUID  `json:"order_id"`
	StripePaymentIntentID string     `json:"stripe_payment_intent_id"`
	Status                string     `json:"status"`
	AmountCents           int64      `json:"amount_cents"`
	Currency              string     `json:"currency"`
	ErrorCode             *string    `json:"error_code,omitempty"`
	ErrorMessage          *string    `json:"error_message,omitempty"`
	StripeRefundID        *string    `json:"stripe_refund_id,omitempty"`
	RefundedAt            *time.Time `json:"refunded_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// CreateIntentResponse is the body returned to the frontend after it asks for
// a PaymentIntent. The client_secret is what Stripe Elements needs to confirm
// the payment; publishable_key is bundled to save a round-trip.
type CreateIntentResponse struct {
	ClientSecret    string `json:"client_secret"`
	PublishableKey  string `json:"publishable_key"`
	PaymentIntentID string `json:"payment_intent_id"`
}
