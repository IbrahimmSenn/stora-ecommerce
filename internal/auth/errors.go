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
)
