// model.go — delivery (shipping) option domain model and request types.
package delivery

import (
	"time"

	"github.com/google/uuid"
)

type DeliveryOption struct {
	ID         uuid.UUID `json:"id"`
	Code       string    `json:"code"`
	Label      string    `json:"label"`
	PriceCents int64     `json:"price_cents"`
	EtaLabel   string    `json:"eta_label"`
	SortOrder  int       `json:"sort_order"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Code       string `json:"code" validate:"required,min=1,max=40"`
	Label      string `json:"label" validate:"required,min=1,max=100"`
	PriceCents int64  `json:"price_cents" validate:"gte=0"`
	EtaLabel   string `json:"eta_label" validate:"max=100"`
	SortOrder  int    `json:"sort_order"`
	Active     *bool  `json:"active,omitempty"`
}

type UpdateRequest struct {
	Label      string `json:"label" validate:"required,min=1,max=100"`
	PriceCents int64  `json:"price_cents" validate:"gte=0"`
	EtaLabel   string `json:"eta_label" validate:"max=100"`
	SortOrder  int    `json:"sort_order"`
	Active     *bool  `json:"active,omitempty"`
}
