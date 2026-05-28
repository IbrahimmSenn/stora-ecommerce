package orders

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPendingPayment = "pending_payment"
	StatusPaid           = "paid"
	StatusPaymentFailed  = "payment_failed"
	StatusProcessing     = "processing"
	StatusShipped        = "shipped"
	StatusDelivered      = "delivered"
	StatusCancelled      = "cancelled"
	StatusRefunded       = "refunded"
)

const (
	ShippingStandard = "standard"
	ShippingExpress  = "express"
)

type Order struct {
	ID             uuid.UUID  `json:"id"`
	OrderNumber    string     `json:"order_number"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	GuestSessionID *uuid.UUID `json:"guest_session_id,omitempty"`
	Status         string     `json:"status"`
	Email          string     `json:"email"`
	Phone          string     `json:"phone,omitempty"`
	SubtotalCents  int64      `json:"subtotal_cents"`
	ShippingCents  int64      `json:"shipping_cents"`
	TotalCents     int64      `json:"total_cents"`
	ShippingMethod string     `json:"shipping_method"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type OrderItem struct {
	ID             uuid.UUID  `json:"id"`
	OrderID        uuid.UUID  `json:"order_id"`
	ProductID      *uuid.UUID `json:"product_id,omitempty"`
	ProductName    string     `json:"product_name"`
	UnitPriceCents int64      `json:"unit_price_cents"`
	Quantity       int        `json:"quantity"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ShippingAddress struct {
	RecipientName string `json:"recipient_name"`
	Line1         string `json:"line1"`
	Line2         string `json:"line2,omitempty"`
	City          string `json:"city"`
	Region        string `json:"region"`
	PostalCode    string `json:"postal_code"`
	Country       string `json:"country"`
}

type OrderResponse struct {
	Order   Order            `json:"order"`
	Items   []OrderItem      `json:"items"`
	Address ShippingAddress  `json:"address"`
}

type OrderSummary struct {
	ID          uuid.UUID `json:"id"`
	OrderNumber string    `json:"order_number"`
	Status      string    `json:"status"`
	TotalCents  int64     `json:"total_cents"`
	ItemCount   int       `json:"item_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// PrefillResponse is what GET /api/v1/checkout/prefill returns for logged-in
// users with at least one prior order. Email and address are pulled from the
// user's most recent order; the frontend uses them to populate the checkout
// form so a returning shopper doesn't retype the same details every time.
type PrefillResponse struct {
	Email          string          `json:"email"`
	Phone          string          `json:"phone,omitempty"`
	ShippingMethod string          `json:"shipping_method"`
	Address        ShippingAddress `json:"address"`
}

// CheckoutRequest is the body for POST /api/v1/checkout.
//
// Addresses are validated for format (length, ISO country) and then for
// deliverability via the injected Geocoder. AddressOverride lets the user
// proceed past a verification failure — the frontend only exposes it after
// the first rejection, so it's a deliberate choice, not a hidden bypass.
type CheckoutRequest struct {
	Email           string                 `json:"email" validate:"required,email"`
	Phone           string                 `json:"phone" validate:"omitempty,min=7,max=20"`
	ShippingMethod  string                 `json:"shipping_method" validate:"required,oneof=standard express"`
	Address         CheckoutAddressRequest `json:"address" validate:"required"`
	AddressOverride bool                   `json:"address_override"`
}

type CheckoutAddressRequest struct {
	RecipientName string `json:"recipient_name" validate:"required,min=1,max=120"`
	Line1         string `json:"line1" validate:"required,min=1,max=200"`
	Line2         string `json:"line2" validate:"omitempty,max=200"`
	City          string `json:"city" validate:"required,min=1,max=120"`
	Region        string `json:"region" validate:"required,min=1,max=120"`
	PostalCode    string `json:"postal_code" validate:"required,min=3,max=12"`
	Country       string `json:"country" validate:"required,len=2,alpha,iso3166_1_alpha2"`
}

// orderRow is the encrypted form returned from the repository. Service
// decrypts to Order before exposing to handlers.
type orderRow struct {
	ID             uuid.UUID
	OrderNumber    string
	UserID         *uuid.UUID
	GuestSessionID *uuid.UUID
	Status         string
	EmailEnc       []byte
	PhoneEnc       []byte
	SubtotalCents  int64
	ShippingCents  int64
	TotalCents     int64
	ShippingMethod string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// addressRow mirrors orderRow — encrypted form for the repository boundary.
type addressRow struct {
	RecipientNameEnc []byte
	Line1Enc         []byte
	Line2Enc         []byte
	CityEnc          []byte
	RegionEnc        []byte
	PostalCodeEnc    []byte
	CountryEnc       []byte
}
