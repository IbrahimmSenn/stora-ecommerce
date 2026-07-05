package orders

import "errors"

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrCartEmpty         = errors.New("cart is empty")
	ErrStockChanged      = errors.New("stock changed during checkout")
	ErrNotCancellable    = errors.New("order cannot be cancelled in its current status")
	ErrForbidden         = errors.New("forbidden")
	ErrNoOwner           = errors.New("checkout requires a logged-in user or a guest session")
	ErrInvalidShipping   = errors.New("invalid shipping method")
	ErrRefundUnavailable = errors.New("refund processor is not configured")
	ErrInvalidStatus     = errors.New("invalid shipping status")
	ErrNotRefundable     = errors.New("order cannot be refunded in its current status")

	// Address verification.
	// ErrAddressNotVerifiable: the geocoder returned no match for the address.
	// ErrAddressVerificationUnavailable: the geocoder itself failed (network,
	// 5xx, rate limit). Distinct so the handler can offer an override flow
	// without blaming the user for a third-party outage.
	ErrAddressNotVerifiable           = errors.New("address could not be verified")
	ErrAddressVerificationUnavailable = errors.New("address verification unavailable")
)
