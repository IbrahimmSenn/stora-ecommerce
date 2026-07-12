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
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Me is the signed-in user's own profile.
type Me struct {
	Id        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateProfileRequest struct {
	// Trimmed before storing; empty clears the name.
	Name string `json:"name"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,max=72"`
}

type UserResponse struct {
	Id    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// AdminUser is a user row for the admin users table.
type AdminUser struct {
	Id        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminUserList wraps the admin users table with pagination metadata.
type AdminUserList struct {
	Users    []AdminUser `json:"users"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// SetRoleRequest is the body for assigning a user's role.
type SetRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin support sales customer"`
}
