package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"
	"github.com/stripe/stripe-go/v76/webhook"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/orders"
)

// IntentClient is the slice of the Stripe SDK we actually call. Tests stub
// this so they don't need network access or real keys.
type IntentClient interface {
	NewIntent(ctx context.Context, amountCents int64, currency string, metadata map[string]string) (id, clientSecret string, err error)
}

// RefundClient wraps the Stripe SDK's refund call. Same reasoning as
// IntentClient: stubbable in unit tests, doesn't leak SDK types upward.
type RefundClient interface {
	Refund(ctx context.Context, paymentIntentID, idempotencyKey string) (refundID string, err error)
}

// stripeIntentClient is the production implementation, backed by stripe-go.
type stripeIntentClient struct{}

func NewStripeClient() IntentClient { return stripeIntentClient{} }

// stripeRefundClient is the production RefundClient.
type stripeRefundClient struct{}

func NewStripeRefundClient() RefundClient { return stripeRefundClient{} }

func (stripeRefundClient) Refund(ctx context.Context, paymentIntentID, idempotencyKey string) (string, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
	}
	params.Context = ctx
	if idempotencyKey != "" {
		params.SetIdempotencyKey(idempotencyKey)
	}
	r, err := refund.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe refund.New: %w", err)
	}
	return r.ID, nil
}

func (stripeIntentClient) NewIntent(ctx context.Context, amount int64, currency string, metadata map[string]string) (string, string, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	for k, v := range metadata {
		params.AddMetadata(k, v)
	}
	params.Context = ctx
	pi, err := paymentintent.New(params)
	if err != nil {
		return "", "", fmt.Errorf("stripe paymentintent.New: %w", err)
	}
	return pi.ID, pi.ClientSecret, nil
}

type Service interface {
	CreateIntent(ctx context.Context, userID, guestID *uuid.UUID, orderID uuid.UUID) (*CreateIntentResponse, error)
	HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error
	RefundOrder(ctx context.Context, orderID uuid.UUID) error
}

type service struct {
	repo           Repository
	orders         orders.Service
	events         EventPublisher
	stripe         IntentClient
	refunds        RefundClient
	webhookSecret  string
	publishableKey string
}

func NewService(repo Repository, ordersSvc orders.Service, events EventPublisher, stripe IntentClient, refunds RefundClient, webhookSecret, publishableKey string) Service {
	return &service{
		repo:           repo,
		orders:         ordersSvc,
		events:         events,
		stripe:         stripe,
		refunds:        refunds,
		webhookSecret:  webhookSecret,
		publishableKey: publishableKey,
	}
}

func (s *service) CreateIntent(ctx context.Context, userID, guestID *uuid.UUID, orderID uuid.UUID) (*CreateIntentResponse, error) {
	order, err := s.orders.GetByID(ctx, userID, guestID, orderID)
	if err != nil {
		if errors.Is(err, orders.ErrForbidden) {
			return nil, ErrForbidden
		}
		return nil, err
	}

	switch order.Order.Status {
	case orders.StatusPendingPayment, orders.StatusPaymentFailed:
		// allowed
	default:
		return nil, ErrInvalidOrderStatus
	}

	intentID, clientSecret, err := s.stripe.NewIntent(ctx, order.Order.TotalCents, "usd", map[string]string{
		"order_id":     order.Order.ID.String(),
		"order_number": order.Order.OrderNumber,
	})
	if err != nil {
		return nil, err
	}

	row := &Payment{
		OrderID:               order.Order.ID,
		StripePaymentIntentID: intentID,
		Status:                StatusPending,
		AmountCents:           order.Order.TotalCents,
		Currency:              "usd",
	}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, err
	}

	return &CreateIntentResponse{
		ClientSecret:    clientSecret,
		PublishableKey:  s.publishableKey,
		PaymentIntentID: intentID,
	}, nil
}

func (s *service) HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	// IgnoreAPIVersionMismatch lets the local stripe-go SDK accept events
	// signed by a Stripe account on a newer API version. Safe for us because
	// we only read pi.ID, pi.LastPaymentError.Code, and pi.LastPaymentError.Msg
	// — fields that have been stable across every API version that ships them.
	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, s.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSignatureMismatch, err)
	}
	return s.handleEvent(ctx, event)
}

// handleEvent is split out from HandleWebhook so unit tests can drive event
// types directly without forging signatures.
func (s *service) handleEvent(ctx context.Context, event stripe.Event) error {
	switch event.Type {
	case "payment_intent.succeeded":
		return s.onIntentSucceeded(ctx, event.Data.Raw)
	case "payment_intent.payment_failed":
		return s.onIntentFailed(ctx, event.Data.Raw)
	}
	// Other events: 200 OK, no-op. Stripe sends many we don't care about.
	return nil
}

func (s *service) onIntentSucceeded(ctx context.Context, raw json.RawMessage) error {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(raw, &pi); err != nil {
		return fmt.Errorf("unmarshal succeeded intent: %w", err)
	}

	existing, err := s.repo.GetByPaymentIntentID(ctx, pi.ID)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			// Unknown intent — likely a `stripe trigger` test event or a
			// retry of one of our pre-fix attempts. Ack so Stripe stops
			// retrying; nothing to do.
			log.Printf("payments: succeeded event for unknown intent %s — ignoring", pi.ID)
			return nil
		}
		return err
	}
	// Idempotency: Stripe retries webhooks. If we already processed this
	// intent, skip the side effects (DB update + email) entirely.
	if existing.Status == StatusSucceeded {
		return nil
	}

	if err := s.repo.UpdateSucceeded(ctx, pi.ID); err != nil {
		return err
	}
	if err := s.orders.MarkPaid(ctx, existing.OrderID); err != nil {
		return err
	}

	evt := PaymentSucceededEvent{
		OrderID:         existing.OrderID,
		PaymentIntentID: pi.ID,
		AmountCents:     existing.AmountCents,
		Currency:        existing.Currency,
	}
	if err := s.events.PublishSucceeded(ctx, evt); err != nil {
		// Don't block Stripe — log and move on. The DB state is correct;
		// the email is best-effort and will be replayed if we add a redrive.
		log.Printf("payments: publish succeeded event failed: %v", err)
	}
	return nil
}

func (s *service) onIntentFailed(ctx context.Context, raw json.RawMessage) error {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(raw, &pi); err != nil {
		return fmt.Errorf("unmarshal failed intent: %w", err)
	}

	existing, err := s.repo.GetByPaymentIntentID(ctx, pi.ID)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			log.Printf("payments: failed event for unknown intent %s — ignoring", pi.ID)
			return nil
		}
		return err
	}
	if existing.Status == StatusFailed {
		return nil
	}

	code, message := "", ""
	if pi.LastPaymentError != nil {
		code = string(pi.LastPaymentError.Code)
		message = pi.LastPaymentError.Msg
	}

	if err := s.repo.UpdateFailed(ctx, pi.ID, code, message); err != nil {
		return err
	}
	if err := s.orders.MarkPaymentFailed(ctx, existing.OrderID); err != nil {
		return err
	}

	evt := PaymentFailedEvent{
		OrderID:         existing.OrderID,
		PaymentIntentID: pi.ID,
		AmountCents:     existing.AmountCents,
		Currency:        existing.Currency,
		FailureCode:     code,
		FailureMessage:  message,
	}
	if err := s.events.PublishFailed(ctx, evt); err != nil {
		log.Printf("payments: publish failed event failed: %v", err)
	}
	return nil
}

// RefundOrder issues a Stripe refund for the latest successful payment on
// the order, then marks the payments row as refunded. Idempotent: if the
// row is already refunded, returns nil without contacting Stripe.
//
// Returns ErrPaymentNotFound if no payment exists, ErrCannotRefund if the
// payment isn't in a refundable state (e.g. never succeeded).
func (s *service) RefundOrder(ctx context.Context, orderID uuid.UUID) error {
	payment, err := s.repo.LatestForOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if payment.Status == StatusRefunded {
		return nil
	}
	if payment.Status != StatusSucceeded {
		return ErrCannotRefund
	}

	// Use the payment row id as the idempotency key so duplicate calls into
	// Stripe don't create duplicate refunds even if our DB write is lost.
	refundID, err := s.refunds.Refund(ctx, payment.StripePaymentIntentID, payment.ID.String())
	if err != nil {
		return fmt.Errorf("stripe refund for order %s: %w", orderID, err)
	}
	return s.repo.MarkRefunded(ctx, payment.ID, refundID)
}
