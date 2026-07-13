package payments

import "errors"

var (
	ErrPaymentNotFound    = errors.New("payment not found")
	ErrInvalidOrderStatus = errors.New("order cannot be paid in its current status")
	ErrForbidden          = errors.New("forbidden")
	ErrSignatureMismatch  = errors.New("stripe webhook signature mismatch")
	ErrOrderNotFound      = errors.New("order not found")
	ErrCannotRefund       = errors.New("payment is not in a refundable state")

	// ErrIntentNotCancellable: Stripe refused to void the intent because it
	// already reached a terminal or in-flight state (succeeded, processing,
	// canceled). CancelOrderIntents follows up with a status check to tell
	// "already dead" apart from "the customer actually paid".
	ErrIntentNotCancellable = errors.New("payment intent is not cancellable")
)
