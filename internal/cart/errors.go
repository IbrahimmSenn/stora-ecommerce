package cart

import "errors"

var (
	ErrCartNotFound = errors.New("cart not found")
	ErrItemNotFound = errors.New("cart item not found")
	ErrOutOfStock   = errors.New("not enough stock available")
	ErrNoOwner      = errors.New("cart requires either a user ID or guest session ID")
)
