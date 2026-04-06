// errors.go — sentinel errors for the user package.
package user

import (
	"errors"
)

var (
	ErrEmailExists  = errors.New("email already taken")
	ErrUserNotFound = errors.New("user not found")
)
