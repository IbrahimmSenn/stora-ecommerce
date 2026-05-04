package payments

import "errors"

var (
	ErrPaymentNotFound    = errors.New("payment not found")
	ErrInvalidOrderStatus = errors.New("order cannot be paid in its current status")
	ErrForbidden          = errors.New("forbidden")
	ErrSignatureMismatch  = errors.New("stripe webhook signature mismatch")
	ErrOrderNotFound      = errors.New("order not found")
)
