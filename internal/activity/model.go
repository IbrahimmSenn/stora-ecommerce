package activity

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventView       = "view"
	EventSearch     = "search"
	EventAddToCart  = "add_to_cart"
	EventPurchase   = "purchase"
)

// Event is a single row in user_activity. Either UserID or GuestSessionID
// must be non-nil. ProductID is required for non-search events; SearchQuery
// is required for search events.
type Event struct {
	ID             int64
	UserID         *uuid.UUID
	GuestSessionID *uuid.UUID
	EventType      string
	ProductID      *uuid.UUID
	CategoryID     *uuid.UUID
	SearchQuery    *string
	OccurredAt     time.Time
}
