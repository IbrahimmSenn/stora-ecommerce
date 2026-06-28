// model.go — category domain model, tree representation, and request types.
package category

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CategoryTree is a category with its nested children, used for browsing.
type CategoryTree struct {
	ID       uuid.UUID      `json:"id"`
	Name     string         `json:"name"`
	Slug     string         `json:"slug"`
	Children []CategoryTree `json:"children,omitempty"`
}

type CreateCategoryRequest struct {
	Name     string  `json:"name" validate:"required,min=1,max=255"`
	Slug     string  `json:"slug" validate:"required,min=1,max=255"`
	ParentID *string `json:"parent_id,omitempty"`
}

type UpdateCategoryRequest struct {
	Name     string  `json:"name" validate:"required,min=1,max=255"`
	Slug     string  `json:"slug" validate:"required,min=1,max=255"`
	ParentID *string `json:"parent_id,omitempty"`
}
