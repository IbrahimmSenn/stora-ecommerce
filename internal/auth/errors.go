package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrNoToken            = errors.New("missing authentication token")
	ErrInvalidToken       = errors.New("invalid or malformed token")
	ErrExpiredToken       = errors.New("token has expired")
	ErrTokenRevoked       = errors.New("token has been revoked")
	ErrTokenUsed          = errors.New("refresh token has already been used")
	ErrTokenNotFound      = errors.New("refresh token not found")
	ErrResetTokenNotFound = errors.New("reset token not found")
	ErrResetTokenUsed     = errors.New("reset token already used")
	ErrResetTokenExpired  = errors.New("reset token has expired")
	Err2FANotEnabled      = errors.New("2fa is not enabled")
	Err2FAAlreadyEnabled  = errors.New("2fa is already enabled")
	ErrInvalid2FACode     = errors.New("invalid 2fa code")
	Err2FARequired        = errors.New("2fa verification required")
)
