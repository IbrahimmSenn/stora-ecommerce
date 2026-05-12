// amqp_publisher.go — concrete EventPublisher backed by RabbitMQ. Marshals
// events to JSON and routes them on the payments topic exchange.
package payments

import (
	"context"
	"encoding/json"
	"fmt"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/messaging"
)

// brokerPublisher is the slice of messaging.Publisher this package needs.
// Lets unit tests stub the broker without standing up rabbitmq.
type brokerPublisher interface {
	Publish(ctx context.Context, exchange, routingKey string, body []byte) error
}

type AmqpPublisher struct {
	pub brokerPublisher
}

func NewAmqpPublisher(pub brokerPublisher) *AmqpPublisher {
	return &AmqpPublisher{pub: pub}
}

func (a *AmqpPublisher) PublishSucceeded(ctx context.Context, evt PaymentSucceededEvent) error {
	body, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal succeeded event: %w", err)
	}
	return a.pub.Publish(ctx, messaging.ExchangePayments, messaging.RoutingKeyPaymentSucceeded, body)
}

func (a *AmqpPublisher) PublishFailed(ctx context.Context, evt PaymentFailedEvent) error {
	body, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal failed event: %w", err)
	}
	return a.pub.Publish(ctx, messaging.ExchangePayments, messaging.RoutingKeyPaymentFailed, body)
}
