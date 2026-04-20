package cart

import "errors"

var (
	ErrCartNotFound = errors.New("cart not found")
	ErrItemNotFound = errors.New("cart item not found")
)
