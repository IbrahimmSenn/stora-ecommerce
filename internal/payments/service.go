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
	GetIntent(ctx context.Context, id string) (IntentStatus, error)
}

// IntentStatus is the trimmed view of a Stripe PaymentIntent used by Reconcile.
// Mirrors only the fields we read so SDK upgrades don't ripple into the service.
type IntentStatus struct {
	Status        string
	LastErrorCode string
	LastErrorMsg  string
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

func (stripeIntentClient) GetIntent(ctx context.Context, id string) (IntentStatus, error) {
	params := &stripe.PaymentIntentParams{}
	params.Context = ctx
	pi, err := paymentintent.Get(id, params)
	if err != nil {
		return IntentStatus{}, fmt.Errorf("stripe paymentintent.Get: %w", err)
	}
	out := IntentStatus{Status: string(pi.Status)}
	if pi.LastPaymentError != nil {
		out.LastErrorCode = string(pi.LastPaymentError.Code)
		out.LastErrorMsg = pi.LastPaymentError.Msg
	}
	return out, nil
}

type Service interface {
	CreateIntent(ctx context.Context, userID, guestID *uuid.UUID, orderID uuid.UUID) (*CreateIntentResponse, error)
	HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error
	RefundOrder(ctx context.Context, orderID uuid.UUID) error

	// Reconcile pulls the current PaymentIntent state from Stripe and applies
	// the same side effects as the corresponding webhook event. Safe to call
	// when no webhook ever arrived; idempotent when one already did.
	Reconcile(ctx context.Context, orderID uuid.UUID) error
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
	return s.applySucceeded(ctx, existing)
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
	return s.applyFailed(ctx, existing, code, message)
}

// applySucceeded runs the side effects for a payment transitioning to
// "succeeded": flip the payment row, mark the order paid, publish the event.
// Caller is responsible for the idempotency check.
func (s *service) applySucceeded(ctx context.Context, existing *Payment) error {
	if err := s.repo.UpdateSucceeded(ctx, existing.StripePaymentIntentID); err != nil {
		return err
	}
	if err := s.orders.MarkPaid(ctx, existing.OrderID); err != nil {
		return err
	}
	evt := PaymentSucceededEvent{
		OrderID:         existing.OrderID,
		PaymentIntentID: existing.StripePaymentIntentID,
		AmountCents:     existing.AmountCents,
		Currency:        existing.Currency,
	}
	if err := s.events.PublishSucceeded(ctx, evt); err != nil {
		// Don't block the caller — the DB state is correct; the email is
		// best-effort and would be replayed by a redrive if we add one.
		log.Printf("payments: publish succeeded event failed: %v", err)
	}
	return nil
}

// applyFailed runs the side effects for a payment transitioning to "failed".
// Caller is responsible for the idempotency check.
func (s *service) applyFailed(ctx context.Context, existing *Payment, code, message string) error {
	if err := s.repo.UpdateFailed(ctx, existing.StripePaymentIntentID, code, message); err != nil {
		return err
	}
	if err := s.orders.MarkPaymentFailed(ctx, existing.OrderID); err != nil {
		return err
	}
	evt := PaymentFailedEvent{
		OrderID:         existing.OrderID,
		PaymentIntentID: existing.StripePaymentIntentID,
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

// Reconcile fetches the current PaymentIntent state from Stripe and applies
// the matching side effects when the order's payment row is still pending.
// This is the safety net for missed webhooks: if the user opens an order
// that's stuck on pending_payment but Stripe says succeeded, this flips it.
func (s *service) Reconcile(ctx context.Context, orderID uuid.UUID) error {
	payment, err := s.repo.LatestForOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			return nil
		}
		return err
	}
	switch payment.Status {
	case StatusSucceeded, StatusFailed, StatusRefunded, StatusCancelled:
		return nil
	}
	intent, err := s.stripe.GetIntent(ctx, payment.StripePaymentIntentID)
	if err != nil {
		return fmt.Errorf("reconcile order %s: %w", orderID, err)
	}
	switch intent.Status {
	case "succeeded":
		return s.applySucceeded(ctx, payment)
	case "canceled":
		return s.applyFailed(ctx, payment, intent.LastErrorCode, intent.LastErrorMsg)
	case "requires_payment_method":
		// Stripe parks a PI here after a failed confirmation. Treat as failed
		// only when there's an explicit error attached — otherwise the user
		// just hasn't finished entering details and the order stays pending.
		if intent.LastErrorCode != "" || intent.LastErrorMsg != "" {
			return s.applyFailed(ctx, payment, intent.LastErrorCode, intent.LastErrorMsg)
		}
		return nil
	}
	// processing, requires_action, requires_confirmation, etc. — leave pending.
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
