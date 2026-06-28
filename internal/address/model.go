// model.go — saved address domain types and request structs.
package address

import (
	"time"

	"github.com/google/uuid"
)

// Address is a user's saved address, decrypted for the API response.
type Address struct {
	ID            uuid.UUID `json:"id"`
	Label         *string   `json:"label,omitempty"`
	RecipientName string    `json:"recipient_name"`
	Line1         string    `json:"line1"`
	Line2         string    `json:"line2,omitempty"`
	City          string    `json:"city"`
	Region        string    `json:"region"`
	PostalCode    string    `json:"postal_code"`
	Country       string    `json:"country"`
	IsDefault     bool      `json:"is_default"`
	CreatedAt     time.Time `json:"created_at"`
}

// AddressRequest is the body for create/update. Mirrors the checkout address
// validation so saved addresses are usable at checkout.
type AddressRequest struct {
	Label         string `json:"label" validate:"omitempty,max=60"`
	RecipientName string `json:"recipient_name" validate:"required,min=1,max=120"`
	Line1         string `json:"line1" validate:"required,min=1,max=200"`
	Line2         string `json:"line2" validate:"omitempty,max=200"`
	City          string `json:"city" validate:"required,min=1,max=120"`
	Region        string `json:"region" validate:"required,min=1,max=120"`
	PostalCode    string `json:"postal_code" validate:"required,min=3,max=12"`
	Country       string `json:"country" validate:"required,len=2,alpha"`
	IsDefault     bool   `json:"is_default"`
}
