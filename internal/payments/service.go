package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"
	"github.com/stripe/stripe-go/v76/webhook"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/metrics"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/orders"
)

// publishRetryBackoffs is the per-attempt delay when the broker is briefly
// unreachable. Keeps the webhook handler bounded (under ~4s in the worst
// case) so Stripe doesn't time out and replay the whole event.
var publishRetryBackoffs = []time.Duration{
	100 * time.Millisecond,
	500 * time.Millisecond,
	2 * time.Second,
}

// publishWithRetry invokes `do` with bounded backoff. The DB side effect is
// already committed by the time we get here, so a final failure must be
// surfaced loudly rather than failing the webhook (Stripe would replay it,
// and our idempotency guard would just no-op the second time).
func publishWithRetry(do func() error, label string) {
	var err error
	for i := 0; i <= len(publishRetryBackoffs); i++ {
		err = do()
		if err == nil {
			if i > 0 {
				log.Printf("payments: %s published after %d retries", label, i)
			}
			return
		}
		if i < len(publishRetryBackoffs) {
			time.Sleep(publishRetryBackoffs[i])
		}
	}
	log.Printf("payments: %s publish failed after %d attempts: %v — manual replay required", label, len(publishRetryBackoffs)+1, err)
}

// IntentClient is the slice of the Stripe SDK we actually call. Tests stub
// this so they don't need network access or real keys.
type IntentClient interface {
	NewIntent(ctx context.Context, amountCents int64, currency string, metadata map[string]string) (id, clientSecret string, err error)
	GetIntent(ctx context.Context, id string) (IntentStatus, error)
	// CancelIntent voids the intent so it can no longer be confirmed. Returns
	// ErrIntentNotCancellable (wrapped) when Stripe rejects the cancel because
	// the intent is in a state that forbids it.
	CancelIntent(ctx context.Context, id string) error
}

// IntentStatus is the trimmed view of a Stripe PaymentIntent used by Reconcile
// and the idempotent CreateIntent path. Mirrors only the fields we read so SDK
// upgrades don't ripple into the service.
type IntentStatus struct {
	Status        string
	LastErrorCode string
	LastErrorMsg  string
	ClientSecret  string
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

func (stripeIntentClient) CancelIntent(ctx context.Context, id string) error {
	params := &stripe.PaymentIntentCancelParams{}
	params.Context = ctx
	_, err := paymentintent.Cancel(id, params)
	if err != nil {
		var sErr *stripe.Error
		if errors.As(err, &sErr) && sErr.Code == stripe.ErrorCodePaymentIntentUnexpectedState {
			return fmt.Errorf("%w: %v", ErrIntentNotCancellable, err)
		}
		return fmt.Errorf("stripe paymentintent.Cancel: %w", err)
	}
	return nil
}

func (stripeIntentClient) GetIntent(ctx context.Context, id string) (IntentStatus, error) {
	params := &stripe.PaymentIntentParams{}
	params.Context = ctx
	pi, err := paymentintent.Get(id, params)
	if err != nil {
		return IntentStatus{}, fmt.Errorf("stripe paymentintent.Get: %w", err)
	}
	out := IntentStatus{Status: string(pi.Status), ClientSecret: pi.ClientSecret}
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

	// CancelOrderIntents voids every still-pending Stripe intent for an order
	// so an abandoned checkout can't be charged after the reaper releases its
	// stock. Returns orders.ErrPaymentInFlight (wrapped) when an intent has
	// already succeeded or is mid-processing — the order isn't abandoned.
	CancelOrderIntents(ctx context.Context, orderID uuid.UUID) error
}

type service struct {
	repo           Repository
	orders         orders.Service
	events         EventPublisher
	stripe         IntentClient
	refunds        RefundClient
	webhookSecret  string
	publishableKey string
	metrics        metrics.Recorder
}

type ServiceOption func(*service)

func WithMetrics(r metrics.Recorder) ServiceOption {
	return func(s *service) { s.metrics = r }
}

func NewService(repo Repository, ordersSvc orders.Service, events EventPublisher, stripe IntentClient, refunds RefundClient, webhookSecret, publishableKey string, opts ...ServiceOption) Service {
	s := &service{
		repo:           repo,
		orders:         ordersSvc,
		events:         events,
		stripe:         stripe,
		refunds:        refunds,
		webhookSecret:  webhookSecret,
		publishableKey: publishableKey,
		metrics:        metrics.Noop{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *service) CreateIntent(ctx context.Context, userID, guestID *uuid.UUID, orderID uuid.UUID) (*CreateIntentResponse, error) {
	order, err := s.orders.GetByID(ctx, userID, guestID, orderID)
	if err != nil {
		if errors.Is(err, orders.ErrForbidden) {
			return nil, ErrForbidden
		}
		return nil, err
	}

	// Only orders that still hold reserved stock can be paid. payment_failed
	// orders had their stock released by MarkPaymentFailed; retrying against
	// the same order would charge the customer without anything to fulfil, so
	// the caller has to start a fresh checkout.
	if order.Order.Status != orders.StatusPendingPayment {
		return nil, ErrInvalidOrderStatus
	}

	// Idempotency: if there's already a pending payment for this order (e.g.
	// the user refreshed the pay page or double-tapped the button), reuse
	// the same Stripe intent rather than creating a duplicate. We fetch the
	// client_secret fresh from Stripe because we never persist it.
	if existing, err := s.repo.LatestForOrder(ctx, order.Order.ID); err == nil && existing.Status == StatusPending {
		intent, err := s.stripe.GetIntent(ctx, existing.StripePaymentIntentID)
		if err == nil && intent.ClientSecret != "" && isReusableIntentStatus(intent.Status) {
			return &CreateIntentResponse{
				ClientSecret:    intent.ClientSecret,
				PublishableKey:  s.publishableKey,
				PaymentIntentID: existing.StripePaymentIntentID,
			}, nil
		}
		// If the existing intent is in a non-reusable state (canceled, succeeded
		// out-of-band) we fall through to create a new one. The orphan row stays
		// in `pending` until Reconcile or a manual sweep tidies it.
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

// isReusableIntentStatus reports whether a Stripe PaymentIntent in this state
// is safe to confirm again with a new card. Intents the user has just opened,
// is mid-3DS, or has had a sync decline against are all reusable.
func isReusableIntentStatus(s string) bool {
	switch s {
	case "requires_payment_method", "requires_confirmation", "requires_action", "processing":
		return true
	}
	return false
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
		s.metrics.PaymentFailed("webhook_signature_invalid")
		slog.Warn("webhook_rejected", "reason", "signature_invalid")
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
// "succeeded". Caller is responsible for the idempotency check.
//
// Order matters: mark the order paid FIRST, flip the payment row LAST. The
// payment row's succeeded status is the idempotency key checked by both the
// webhook handler and Reconcile, so it must only be set once everything else is
// durably done. If MarkPaid succeeds but UpdateSucceeded fails, the guard stays
// open and the next webhook retry (or a Reconcile) re-runs both — MarkPaid is a
// no-op the second time. This closes the "charged order stuck in
// pending_payment forever" hole where the two writes were transposed.
func (s *service) applySucceeded(ctx context.Context, existing *Payment) error {
	paid, err := s.orders.MarkPaid(ctx, existing.OrderID)
	if err != nil {
		return err
	}
	if !paid {
		// Charged, but the order is no longer pending — e.g. the checkout
		// reaper cancelled it before a late payment landed. Record the payment
		// truthfully, but don't confirm the purchase: no confirmation email,
		// no revenue metric. Auto-refund; if that fails, the log line is the
		// signal for a manual refund in the Stripe dashboard.
		slog.Error("payment_succeeded_for_dead_order",
			"order_id", existing.OrderID,
			"payment_intent_id", existing.StripePaymentIntentID,
			"amount_cents", existing.AmountCents)
		if err := s.repo.UpdateSucceeded(ctx, existing.StripePaymentIntentID); err != nil {
			return err
		}
		s.metrics.PaymentOrphaned()
		if err := s.RefundOrder(ctx, existing.OrderID); err != nil {
			slog.Error("auto-refund of orphaned payment failed — refund manually in the Stripe dashboard",
				"order_id", existing.OrderID,
				"payment_intent_id", existing.StripePaymentIntentID,
				"err", err)
		} else {
			slog.Info("orphaned payment auto-refunded", "order_id", existing.OrderID)
		}
		return nil
	}
	if err := s.repo.UpdateSucceeded(ctx, existing.StripePaymentIntentID); err != nil {
		return err
	}
	s.metrics.PaymentSucceeded()
	s.metrics.OrderPaid(existing.AmountCents)
	evt := PaymentSucceededEvent{
		OrderID:         existing.OrderID,
		PaymentIntentID: existing.StripePaymentIntentID,
		AmountCents:     existing.AmountCents,
		Currency:        existing.Currency,
	}
	publishWithRetry(func() error {
		return s.events.PublishSucceeded(ctx, evt)
	}, "payment.succeeded")
	return nil
}

// failureReason maps a Stripe error code onto the bounded label set used by
// shop_payments_total — unknown codes collapse to "other" so cardinality
// stays fixed no matter what Stripe sends.
func failureReason(code string) string {
	switch code {
	case "card_declined", "insufficient_funds", "expired_card", "incorrect_cvc",
		"incorrect_number", "processing_error", "authentication_required":
		return code
	case "":
		return "unknown"
	default:
		return "other"
	}
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
	s.metrics.PaymentFailed(failureReason(code))
	evt := PaymentFailedEvent{
		OrderID:         existing.OrderID,
		PaymentIntentID: existing.StripePaymentIntentID,
		AmountCents:     existing.AmountCents,
		Currency:        existing.Currency,
		FailureCode:     code,
		FailureMessage:  message,
	}
	publishWithRetry(func() error {
		return s.events.PublishFailed(ctx, evt)
	}, "payment.failed")
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

// CancelOrderIntents voids the order's pending Stripe intents (see the
// Service interface). Rows are flipped to cancelled only after Stripe
// confirms the void, and only from pending, so a webhook that raced us can't
// be overwritten.
func (s *service) CancelOrderIntents(ctx context.Context, orderID uuid.UUID) error {
	intentIDs, err := s.repo.PendingIntentIDsForOrder(ctx, orderID)
	if err != nil {
		return err
	}
	for _, intentID := range intentIDs {
		switch err := s.stripe.CancelIntent(ctx, intentID); {
		case err == nil:
		case errors.Is(err, ErrIntentNotCancellable):
			// The intent moved past cancellable on Stripe's side. Succeeded or
			// processing means the customer actually paid — the caller must not
			// treat the order as abandoned. Anything else (canceled out-of-band)
			// is dead and safe to record as cancelled.
			st, gerr := s.stripe.GetIntent(ctx, intentID)
			if gerr != nil {
				return fmt.Errorf("intent %s not cancellable, status check failed: %w", intentID, gerr)
			}
			if st.Status == "succeeded" || st.Status == "processing" {
				return fmt.Errorf("intent %s is %s: %w", intentID, st.Status, orders.ErrPaymentInFlight)
			}
		default:
			return err
		}
		if err := s.repo.UpdateCancelled(ctx, intentID); err != nil {
			return err
		}
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
