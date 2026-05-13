// model.go — request/response types and DB models for auth, tokens, password resets, and 2FA.
package auth

import (
	"time"

	"github.com/google/uuid"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	TOTPCode string `json:"totp_code,omitempty"`
}

type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

type RefreshRequest struct {
	// Optional in the JSON body: when the refresh_token HttpOnly cookie is
	// present the handler reads it from there and populates this field before
	// calling the service. The service still requires a non-empty value.
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=NewPassword"`
}

type AuthMessageResponse struct {
	Message string `json:"message"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"id"`
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	Revoked   bool      `json:"revoked"`
	Used      bool      `json:"used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Password reset ---

type PasswordResetToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"token"`
	Used      bool      `json:"used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// --- 2FA ---

type TwoFactorAuth struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"user_id"`
	SecretKey     string    `json:"-"`
	IsEnabled     bool      `json:"is_enabled"`
	RecoveryCodes []string  `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
}

type Setup2FAResponse struct {
	Secret        string   `json:"secret"`
	QRCode        string   `json:"qr_code"` // base64-encoded PNG
	RecoveryCodes []string `json:"recovery_codes"`
}

type Verify2FARequest struct {
	Code string `json:"code" validate:"required"`
}

type LoginWith2FARequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	TOTPCode string `json:"totp_code" validate:"required"`
}
