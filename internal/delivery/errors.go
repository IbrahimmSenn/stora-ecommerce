// errors.go — sentinel errors for the delivery package.
package delivery

import "errors"

var (
	ErrNotFound    = errors.New("delivery option not found")
	ErrCodeExists  = errors.New("delivery option code already exists")
	ErrInvalidCode = errors.New("code must be lowercase letters, digits, and hyphens")
)
