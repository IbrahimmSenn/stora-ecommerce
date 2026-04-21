package cart

import (
	"time"

	"github.com/google/uuid"
)

type Cart struct {
	ID             uuid.UUID  `json:"id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	GuestSessionID *uuid.UUID `json:"guest_session_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CartItem struct {
	ID        uuid.UUID `json:"id"`
	CartID    uuid.UUID `json:"cart_id"`
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CartItemDetail includes product info for display.
type CartItemDetail struct {
	ID           uuid.UUID `json:"id"`
	ProductID    uuid.UUID `json:"product_id"`
	ProductName  string    `json:"product_name"`
	ProductPrice int64     `json:"product_price"`
	ImageURL     *string   `json:"image_url,omitempty"`
	Quantity     int       `json:"quantity"`
	Stock        int       `json:"stock"`
}

// CartResponse is the full cart with items and total.
type CartResponse struct {
	ID    uuid.UUID        `json:"id"`
	Items []CartItemDetail `json:"items"`
	Total int64            `json:"total"`
}

type AddItemRequest struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,min=1,max=99"`
}

type UpdateItemRequest struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,min=1,max=99"`
}
