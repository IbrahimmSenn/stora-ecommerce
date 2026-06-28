// model.go — review domain types, request/response structs, sort and status enums.
package review

import (
	"time"

	"github.com/google/uuid"
)

// Moderation states. New reviews start Pending; only Approved reviews are
// shown publicly and counted toward a product's average rating.
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusHidden   = "hidden"
)

// Public sort options for a product's review list.
const (
	SortHelpful = "helpful" // default
	SortNewest  = "newest"
	SortHighest = "highest"
	SortLowest  = "lowest"
)

// Review is the core domain model.
type Review struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	ProductID uuid.UUID `json:"product_id"`
	Rating    int       `json:"rating"`
	Comment   *string   `json:"comment,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PublicReview is a review as shown on the product detail page. Reviewers are
// labelled "Verified buyer" rather than by name — the platform collects no
// display name, and this keeps reviewer PII out of the public payload.
type PublicReview struct {
	ID           uuid.UUID `json:"id"`
	Rating       int       `json:"rating"`
	Comment      *string   `json:"comment,omitempty"`
	HelpfulCount int       `json:"helpful_count"`
	VotedByMe    bool      `json:"voted_by_me"`
	MineToEdit   bool      `json:"mine_to_edit"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListResult wraps a product's public reviews with the rating breakdown and
// pagination metadata.
type ListResult struct {
	Reviews      []PublicReview `json:"reviews"`
	Total        int            `json:"total"`
	Page         int            `json:"page"`
	PageSize     int            `json:"page_size"`
	AvgRating    float64        `json:"avg_rating"`
	Distribution map[int]int    `json:"distribution"` // star (1-5) -> count
}

// Eligibility tells the frontend whether the current user may write a review.
type Eligibility struct {
	CanReview       bool `json:"can_review"`
	HasPurchased    bool `json:"has_purchased"`
	AlreadyReviewed bool `json:"already_reviewed"`
	ExistingRating  *int `json:"existing_rating,omitempty"`
	ExistingPending bool `json:"existing_pending"`
}

// ModerationItem is a review row for the admin moderation queue, with product
// context attached.
type ModerationItem struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Rating      int       `json:"rating"`
	Comment     *string   `json:"comment,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// --- Request types ---

type CreateReviewRequest struct {
	Rating  int     `json:"rating" validate:"required,min=1,max=5"`
	Comment *string `json:"comment,omitempty" validate:"omitempty,max=2000"`
}

type ListParams struct {
	ProductID string
	Sort      string
	Page      int
	PageSize  int
	ViewerID  *uuid.UUID // current user, for voted_by_me / mine_to_edit flags
}
