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
	"github.com/stripe/stripe-go/v76/webhook"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/orders"
)

// Mailer is the slice of internal/mailer.Mailer the service uses. Defined
// here (consumer-side) so tests can stub it without touching the mailer
// package.
type Mailer interface {
	Send(to, subject, body string) error
}

// IntentClient is the slice of the Stripe SDK we actually call. Tests stub
// this so they don't need network access or real keys.
type IntentClient interface {
	NewIntent(ctx context.Context, amountCents int64, currency string, metadata map[string]string) (id, clientSecret string, err error)
}

// stripeIntentClient is the production implementation, backed by stripe-go.
type stripeIntentClient struct{}

func NewStripeClient() IntentClient { return stripeIntentClient{} }

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
}

type service struct {
	repo            Repository
	orders          orders.Service
	mail            Mailer
	stripe          IntentClient
	webhookSecret   string
	publishableKey  string
}

func NewService(repo Repository, ordersSvc orders.Service, mail Mailer, stripe IntentClient, webhookSecret, publishableKey string) Service {
	return &service{
		repo:           repo,
		orders:         ordersSvc,
		mail:           mail,
		stripe:         stripe,
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
	event, err := webhook.ConstructEvent(payload, sigHeader, s.webhookSecret)
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

	// TODO(rabbitmq): replace this synchronous send with a published event
	// once the messaging milestone lands.
	s.sendConfirmationEmail(ctx, existing.OrderID)
	return nil
}

func (s *service) onIntentFailed(ctx context.Context, raw json.RawMessage) error {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(raw, &pi); err != nil {
		return fmt.Errorf("unmarshal failed intent: %w", err)
	}

	existing, err := s.repo.GetByPaymentIntentID(ctx, pi.ID)
	if err != nil {
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

	// TODO(rabbitmq): replace this synchronous send with a published event.
	s.sendFailureEmail(ctx, existing.OrderID, code, message)
	return nil
}

func (s *service) sendConfirmationEmail(ctx context.Context, orderID uuid.UUID) {
	order, err := s.orders.LoadByID(ctx, orderID)
	if err != nil {
		log.Printf("payments: load order for confirmation email failed: %v", err)
		return
	}
	subject := fmt.Sprintf("Order %s — payment received", order.Order.OrderNumber)
	body := buildConfirmationBody(order)
	if err := s.mail.Send(order.Order.Email, subject, body); err != nil {
		log.Printf("payments: send confirmation email failed: %v", err)
	}
}

func (s *service) sendFailureEmail(ctx context.Context, orderID uuid.UUID, code, message string) {
	order, err := s.orders.LoadByID(ctx, orderID)
	if err != nil {
		log.Printf("payments: load order for failure email failed: %v", err)
		return
	}
	subject := fmt.Sprintf("Order %s — payment failed", order.Order.OrderNumber)
	body := buildFailureBody(order, code, message)
	if err := s.mail.Send(order.Order.Email, subject, body); err != nil {
		log.Printf("payments: send failure email failed: %v", err)
	}
}

func buildConfirmationBody(o *orders.OrderResponse) string {
	itemsHTML := ""
	for _, it := range o.Items {
		itemsHTML += fmt.Sprintf(
			"<tr><td>%s × %d</td><td style='text-align:right'>$%.2f</td></tr>",
			it.ProductName, it.Quantity, float64(it.UnitPriceCents*int64(it.Quantity))/100,
		)
	}
	return fmt.Sprintf(`<p>Thanks for your order.</p>
<p><strong>Order %s</strong></p>
<table style="border-collapse:collapse">%s
<tr><td>Shipping</td><td style="text-align:right">$%.2f</td></tr>
<tr><td><strong>Total</strong></td><td style="text-align:right"><strong>$%.2f</strong></td></tr>
</table>
<p>Shipping to %s, %s %s.</p>`,
		o.Order.OrderNumber, itemsHTML,
		float64(o.Order.ShippingCents)/100,
		float64(o.Order.TotalCents)/100,
		o.Address.City, o.Address.Region, o.Address.Country,
	)
}

func buildFailureBody(o *orders.OrderResponse, code, message string) string {
	reason := message
	if reason == "" {
		reason = "Your card was declined."
	}
	codeLine := ""
	if code != "" {
		codeLine = fmt.Sprintf("<p style='color:#666'>Reference: %s</p>", code)
	}
	return fmt.Sprintf(`<p>We couldn't process payment for order <strong>%s</strong>.</p>
<p>%s</p>%s
<p>You can retry from your order detail page.</p>`,
		o.Order.OrderNumber, reason, codeLine,
	)
}
