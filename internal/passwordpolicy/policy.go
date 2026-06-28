// Package passwordpolicy defines the shared password-strength rules enforced at
// registration and password reset. The frontend mirrors these rules in a live
// checklist (web/src/auth/passwordCriteria.ts); keep the two in sync.
package passwordpolicy

import (
	"errors"
	"unicode"
)

const (
	// MinLength is the minimum password length.
	MinLength = 8
	// MaxLength caps input at bcrypt's 72-byte limit.
	MaxLength = 72
)

// ErrWeak is returned when a password fails any strength rule. The message
// lists every requirement so the API response is actionable on its own.
var ErrWeak = errors.New(
	"password must be at least 8 characters and include an uppercase letter, a lowercase letter, a number, and a symbol",
)

// Validate returns nil when pw satisfies every rule, otherwise ErrWeak.
func Validate(pw string) error {
	if len(pw) < MinLength || len(pw) > MaxLength {
		return ErrWeak
	}
	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	if hasUpper && hasLower && hasDigit && hasSymbol {
		return nil
	}
	return ErrWeak
}
