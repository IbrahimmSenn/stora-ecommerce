package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/messaging"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/orders"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/payments"
)

type stubLoader struct {
	order *orders.OrderResponse
	err   error
}

func (s *stubLoader) LoadByID(_ context.Context, _ uuid.UUID) (*orders.OrderResponse, error) {
	return s.order, s.err
}

type stubMailer struct {
	calls []sent
	err   error
}

type sent struct {
	to      string
	subject string
	body    string
}

func (m *stubMailer) Send(to, subject, body string) error {
	m.calls = append(m.calls, sent{to, subject, body})
	return m.err
}

func makeOrder(id uuid.UUID) *orders.OrderResponse {
	return &orders.OrderResponse{
		Order: orders.Order{
			ID: id, OrderNumber: "ORD-TEST",
			Email:          "buyer@example.com",
			TotalCents:     2500,
			ShippingCents:  500,
			SubtotalCents:  2000,
			ShippingMethod: "standard",
		},
		Items: []orders.OrderItem{{
			ID: uuid.New(), ProductName: "Widget", UnitPriceCents: 2000, Quantity: 1,
		}},
		Address: orders.ShippingAddress{
			RecipientName: "Buyer", Line1: "1 Demo", City: "X", Region: "Y", PostalCode: "0", Country: "US",
		},
	}
}

func TestEmailConsumer_SucceededRoutesToConfirmation(t *testing.T) {
	orderID := uuid.New()
	mail := &stubMailer{}
	c := &EmailConsumer{Orders: &stubLoader{order: makeOrder(orderID)}, Mail: mail}

	body, _ := json.Marshal(payments.PaymentSucceededEvent{
		OrderID: orderID, PaymentIntentID: "pi_x", AmountCents: 2500, Currency: "usd",
	})
	require.NoError(t, c.Handle(context.Background(), messaging.RoutingKeyPaymentSucceeded, body))

	require.Len(t, mail.calls, 1)
	assert.Equal(t, "buyer@example.com", mail.calls[0].to)
	assert.Contains(t, mail.calls[0].subject, "ORD-TEST")
	assert.Contains(t, mail.calls[0].subject, "received")
	assert.Contains(t, mail.calls[0].body, "Widget")
}

func TestEmailConsumer_FailedIncludesReasonAndCode(t *testing.T) {
	orderID := uuid.New()
	mail := &stubMailer{}
	c := &EmailConsumer{Orders: &stubLoader{order: makeOrder(orderID)}, Mail: mail}

	body, _ := json.Marshal(payments.PaymentFailedEvent{
		OrderID: orderID, PaymentIntentID: "pi_z", AmountCents: 2500, Currency: "usd",
		FailureCode: "card_declined", FailureMessage: "insufficient funds",
	})
	require.NoError(t, c.Handle(context.Background(), messaging.RoutingKeyPaymentFailed, body))

	require.Len(t, mail.calls, 1)
	assert.Contains(t, mail.calls[0].subject, "failed")
	assert.Contains(t, mail.calls[0].body, "insufficient funds")
	assert.Contains(t, mail.calls[0].body, "card_declined")
}

func TestEmailConsumer_FailedWithoutCodeOmitsReferenceLine(t *testing.T) {
	orderID := uuid.New()
	mail := &stubMailer{}
	c := &EmailConsumer{Orders: &stubLoader{order: makeOrder(orderID)}, Mail: mail}

	body, _ := json.Marshal(payments.PaymentFailedEvent{
		OrderID: orderID, PaymentIntentID: "pi_z",
	})
	require.NoError(t, c.Handle(context.Background(), messaging.RoutingKeyPaymentFailed, body))

	require.Len(t, mail.calls, 1)
	assert.Contains(t, mail.calls[0].body, "Your card was declined.")
	assert.NotContains(t, mail.calls[0].body, "Reference:")
}

func TestEmailConsumer_UnknownRoutingKey(t *testing.T) {
	c := &EmailConsumer{Orders: &stubLoader{}, Mail: &stubMailer{}}
	err := c.Handle(context.Background(), "payment.surprise", []byte(`{}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown routing key")
}

func TestEmailConsumer_MalformedJSON(t *testing.T) {
	c := &EmailConsumer{Orders: &stubLoader{}, Mail: &stubMailer{}}
	err := c.Handle(context.Background(), messaging.RoutingKeyPaymentSucceeded, []byte(`not json`))
	require.Error(t, err)
}

func TestEmailConsumer_MailerErrorPropagates(t *testing.T) {
	orderID := uuid.New()
	mail := &stubMailer{err: errors.New("smtp down")}
	c := &EmailConsumer{Orders: &stubLoader{order: makeOrder(orderID)}, Mail: mail}

	body, _ := json.Marshal(payments.PaymentSucceededEvent{OrderID: orderID})
	err := c.Handle(context.Background(), messaging.RoutingKeyPaymentSucceeded, body)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp down")
}

func TestEmailConsumer_OrderLoadErrorPropagates(t *testing.T) {
	mail := &stubMailer{}
	c := &EmailConsumer{Orders: &stubLoader{err: errors.New("db down")}, Mail: mail}

	body, _ := json.Marshal(payments.PaymentSucceededEvent{OrderID: uuid.New()})
	err := c.Handle(context.Background(), messaging.RoutingKeyPaymentSucceeded, body)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
	assert.Empty(t, mail.calls)
}
