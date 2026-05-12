// email_consumer.go — handler for messages on the payments.emails queue.
// Decodes a payment event, loads the order, renders an email body, and
// sends via the mailer. Returning an error here causes the messaging
// consumer to retry (with backoff) and ultimately route to the DLX.
package notifications

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/messaging"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/orders"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/payments"
)

// OrderLoader is the slice of orders.Service the consumer needs. Defined
// here so tests don't need a full orders service.
type OrderLoader interface {
	LoadByID(ctx context.Context, id uuid.UUID) (*orders.OrderResponse, error)
}

// Mailer is the slice of the SMTP sender the consumer calls.
type Mailer interface {
	Send(to, subject, body string) error
}

type EmailConsumer struct {
	Orders OrderLoader
	Mail   Mailer
}

func (c *EmailConsumer) Handle(ctx context.Context, routingKey string, body []byte) error {
	switch routingKey {
	case messaging.RoutingKeyPaymentSucceeded:
		var evt payments.PaymentSucceededEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			return fmt.Errorf("decode succeeded event: %w", err)
		}
		return c.sendConfirmation(ctx, evt)

	case messaging.RoutingKeyPaymentFailed:
		var evt payments.PaymentFailedEvent
		if err := json.Unmarshal(body, &evt); err != nil {
			return fmt.Errorf("decode failed event: %w", err)
		}
		return c.sendFailure(ctx, evt)

	default:
		return fmt.Errorf("unknown routing key: %s", routingKey)
	}
}

func (c *EmailConsumer) sendConfirmation(ctx context.Context, evt payments.PaymentSucceededEvent) error {
	order, err := c.Orders.LoadByID(ctx, evt.OrderID)
	if err != nil {
		return fmt.Errorf("load order %s: %w", evt.OrderID, err)
	}
	subject := fmt.Sprintf("Order %s — payment received", order.Order.OrderNumber)
	body := buildConfirmationBody(order)
	if err := c.Mail.Send(order.Order.Email, subject, body); err != nil {
		return fmt.Errorf("send confirmation to %s: %w", order.Order.Email, err)
	}
	return nil
}

func (c *EmailConsumer) sendFailure(ctx context.Context, evt payments.PaymentFailedEvent) error {
	order, err := c.Orders.LoadByID(ctx, evt.OrderID)
	if err != nil {
		return fmt.Errorf("load order %s: %w", evt.OrderID, err)
	}
	subject := fmt.Sprintf("Order %s — payment failed", order.Order.OrderNumber)
	body := buildFailureBody(order, evt.FailureCode, evt.FailureMessage)
	if err := c.Mail.Send(order.Order.Email, subject, body); err != nil {
		return fmt.Errorf("send failure to %s: %w", order.Order.Email, err)
	}
	return nil
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
