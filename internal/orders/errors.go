package orders

import "errors"

var (
	ErrOrderNotFound  = errors.New("order not found")
	ErrCartEmpty      = errors.New("cart is empty")
	ErrStockChanged   = errors.New("stock changed during checkout")
	ErrNotCancellable = errors.New("order cannot be cancelled in its current status")
	ErrForbidden      = errors.New("forbidden")
	ErrNoOwner        = errors.New("checkout requires a logged-in user or a guest session")
	ErrInvalidShipping = errors.New("invalid shipping method")
)
