// topology.go — exchange + queue declarations for the payments event bus.
// Declared idempotently at boot by both the publisher and consumer wiring
// so either side can start first.
package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangePayments    = "payments"
	ExchangePaymentsDLX = "payments.dlx"

	QueuePaymentEmails    = "payments.emails"
	QueuePaymentEmailsDLQ = "payments.emails.dlq"

	RoutingKeyPaymentSucceeded = "payment.succeeded"
	RoutingKeyPaymentFailed    = "payment.failed"
	BindingKeyPaymentAll       = "payment.*"
)

// DeclarePaymentsTopology declares the payments exchange + DLX, the emails
// queue bound to payment.*, and the dead-letter queue. Safe to call from
// multiple processes; AMQP declares are idempotent when arguments match.
func DeclarePaymentsTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(ExchangePayments, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange %s: %w", ExchangePayments, err)
	}
	if err := ch.ExchangeDeclare(ExchangePaymentsDLX, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange %s: %w", ExchangePaymentsDLX, err)
	}

	if _, err := ch.QueueDeclare(QueuePaymentEmails, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange": ExchangePaymentsDLX,
	}); err != nil {
		return fmt.Errorf("declare queue %s: %w", QueuePaymentEmails, err)
	}
	if err := ch.QueueBind(QueuePaymentEmails, BindingKeyPaymentAll, ExchangePayments, false, nil); err != nil {
		return fmt.Errorf("bind queue %s: %w", QueuePaymentEmails, err)
	}

	if _, err := ch.QueueDeclare(QueuePaymentEmailsDLQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue %s: %w", QueuePaymentEmailsDLQ, err)
	}
	if err := ch.QueueBind(QueuePaymentEmailsDLQ, "", ExchangePaymentsDLX, false, nil); err != nil {
		return fmt.Errorf("bind queue %s: %w", QueuePaymentEmailsDLQ, err)
	}
	return nil
}
