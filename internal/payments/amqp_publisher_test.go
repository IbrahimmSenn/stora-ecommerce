package payments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/messaging"
)

type fakeBroker struct {
	calls []brokerCall
	err   error
}

type brokerCall struct {
	exchange   string
	routingKey string
	body       []byte
}

func (f *fakeBroker) Publish(_ context.Context, exchange, routingKey string, body []byte) error {
	f.calls = append(f.calls, brokerCall{exchange, routingKey, body})
	return f.err
}

func TestAmqpPublisher_Succeeded(t *testing.T) {
	broker := &fakeBroker{}
	pub := NewAmqpPublisher(broker)
	orderID := uuid.New()

	err := pub.PublishSucceeded(context.Background(), PaymentSucceededEvent{
		OrderID:         orderID,
		PaymentIntentID: "pi_abc",
		AmountCents:     2500,
		Currency:        "usd",
	})
	require.NoError(t, err)
	require.Len(t, broker.calls, 1)

	c := broker.calls[0]
	assert.Equal(t, messaging.ExchangePayments, c.exchange)
	assert.Equal(t, messaging.RoutingKeyPaymentSucceeded, c.routingKey)

	var decoded PaymentSucceededEvent
	require.NoError(t, json.Unmarshal(c.body, &decoded))
	assert.Equal(t, orderID, decoded.OrderID)
	assert.Equal(t, "pi_abc", decoded.PaymentIntentID)
	assert.Equal(t, int64(2500), decoded.AmountCents)
}

func TestAmqpPublisher_Failed(t *testing.T) {
	broker := &fakeBroker{}
	pub := NewAmqpPublisher(broker)
	orderID := uuid.New()

	err := pub.PublishFailed(context.Background(), PaymentFailedEvent{
		OrderID:         orderID,
		PaymentIntentID: "pi_zzz",
		AmountCents:     2500,
		Currency:        "usd",
		FailureCode:     "card_declined",
		FailureMessage:  "insufficient funds",
	})
	require.NoError(t, err)
	require.Len(t, broker.calls, 1)

	c := broker.calls[0]
	assert.Equal(t, messaging.RoutingKeyPaymentFailed, c.routingKey)

	var decoded PaymentFailedEvent
	require.NoError(t, json.Unmarshal(c.body, &decoded))
	assert.Equal(t, "card_declined", decoded.FailureCode)
	assert.Equal(t, "insufficient funds", decoded.FailureMessage)
}

func TestAmqpPublisher_PropagatesBrokerError(t *testing.T) {
	broker := &fakeBroker{err: errors.New("broker down")}
	pub := NewAmqpPublisher(broker)
	err := pub.PublishSucceeded(context.Background(), PaymentSucceededEvent{OrderID: uuid.New()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broker down")
}
