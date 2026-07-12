// errors.go — sentinel errors for the user package.
package user

import (
	"errors"
)

var (
	ErrEmailExists    = errors.New("email already taken")
	ErrUserNotFound   = errors.New("user not found")
	ErrCaptchaInvalid = errors.New("captcha verification failed")
	ErrInvalidRole    = errors.New("invalid role")
	ErrLastAdmin      = errors.New("cannot remove the last admin account")
	ErrWrongPassword  = errors.New("current password is incorrect")
	ErrNoPassword     = errors.New("this account signs in with Google or Facebook and has no password to change")
	ErrNameTooLong    = errors.New("name must be 100 characters or fewer")
)
