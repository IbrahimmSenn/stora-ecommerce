// model.go — user domain model and request/response types.
package user

import (
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email        string `json:"email" validate:"required,email"`
	Password     string `json:"password" validate:"required,min=8,max=72"`
	CaptchaToken string `json:"captcha_token,omitempty"`
}
type User struct {
	Id           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserResponse struct {
	Id    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}
