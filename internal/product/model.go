// model.go — product types, request/response structs, and search params.
package product

import (
	"time"

	"github.com/google/uuid"
)

// Product is the core domain model.
type Product struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Description    *string    `json:"description,omitempty"`
	Price          int64      `json:"price"` // cents
	SalePrice      *int64     `json:"sale_price,omitempty"` // cents; nil when not on sale
	StockQuantity  int        `json:"stock_quantity"`
	CategoryID     *uuid.UUID `json:"category_id,omitempty"`
	BrandID        *uuid.UUID `json:"brand_id,omitempty"`
	WeightG        *int       `json:"weight_g,omitempty"`
	WeightOz       *int       `json:"weight_oz,omitempty"`
	DimensionsCm   *float64   `json:"dimensions_cm,omitempty"`
	DimensionsInch *float64   `json:"dimensions_inch,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ProductDetail is a product enriched with its relations for the detail endpoint.
type ProductDetail struct {
	Product
	CategoryName *string        `json:"category_name,omitempty"`
	CategorySlug *string        `json:"category_slug,omitempty"`
	BrandName    *string        `json:"brand_name,omitempty"`
	AvgRating    float64        `json:"avg_rating"`
	ReviewCount  int            `json:"review_count"`
	Images       []ProductImage `json:"images"`
}

// ProductListItem is the compact version returned in search/list results.
type ProductListItem struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Price         int64     `json:"price"`
	SalePrice     *int64    `json:"sale_price,omitempty"`
	StockQuantity int       `json:"stock_quantity"`
	CategoryName  *string   `json:"category_name,omitempty"`
	BrandName     *string   `json:"brand_name,omitempty"`
	AvgRating     float64   `json:"avg_rating"`
	ReviewCount   int       `json:"review_count"`
	PrimaryImage  *string   `json:"primary_image,omitempty"`
	Relevance     *float64  `json:"relevance,omitempty"`
}

type ProductImage struct {
	ID           uuid.UUID `json:"id"`
	ProductID    uuid.UUID `json:"product_id"`
	URL          string    `json:"url"`
	ThumbnailURL *string   `json:"thumbnail_url,omitempty"`
	CardURL      *string   `json:"card_url,omitempty"`
	FullURL      *string   `json:"full_url,omitempty"`
	IsPrimary    bool      `json:"is_primary"`
}

// --- Request types ---

type CreateProductRequest struct {
	Name          string   `json:"name" validate:"required,min=1,max=255"`
	Description   *string  `json:"description,omitempty"`
	Price         int64    `json:"price" validate:"gte=0"`
	SalePrice     *int64   `json:"sale_price,omitempty" validate:"omitempty,gte=0"`
	StockQuantity int      `json:"stock_quantity" validate:"gte=0"`
	CategoryID    *string  `json:"category_id,omitempty"`
	BrandID       *string  `json:"brand_id,omitempty"`
	WeightG       *int     `json:"weight_g,omitempty" validate:"omitempty,gte=0"`
	DimensionsCm  *float64 `json:"dimensions_cm,omitempty" validate:"omitempty,gte=0"`
}

type UpdateProductRequest struct {
	Name          *string  `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description   *string  `json:"description,omitempty"`
	Price         *int64   `json:"price,omitempty" validate:"omitempty,gte=0"`
	SalePrice     *int64   `json:"sale_price,omitempty" validate:"omitempty,gte=0"`
	ClearSalePrice bool    `json:"clear_sale_price,omitempty"`
	StockQuantity *int     `json:"stock_quantity,omitempty" validate:"omitempty,gte=0"`
	CategoryID    *string  `json:"category_id,omitempty"`
	BrandID       *string  `json:"brand_id,omitempty"`
	WeightG       *int     `json:"weight_g,omitempty" validate:"omitempty,gte=0"`
	DimensionsCm  *float64 `json:"dimensions_cm,omitempty" validate:"omitempty,gte=0"`
}

type AddImageRequest struct {
	URL       string `json:"url" validate:"required,url"`
	IsPrimary bool   `json:"is_primary"`
}

// BulkItemError reports why a single row in a bulk upload failed.
type BulkItemError struct {
	Index int    `json:"index"`
	Name  string `json:"name,omitempty"`
	Error string `json:"error"`
}

// BulkResult summarises a bulk product upload (JSON or CSV).
type BulkResult struct {
	Created int             `json:"created"`
	Failed  int             `json:"failed"`
	Errors  []BulkItemError `json:"errors"`
}

// SearchParams holds all faceted search and pagination parameters.
type SearchParams struct {
	Query      string  // free-text search
	CategoryID *string // filter by category
	BrandID    *string // filter by brand
	MinPrice   *int64  // filter by minimum price (cents)
	MaxPrice   *int64  // filter by maximum price (cents)
	MinRating  *int    // filter by minimum average rating
	OnSale     bool    // when true, only products with a sale_price
	SortBy     string  // "relevance", "price_asc", "price_desc", "rating", "discount"
	Page       int
	PageSize   int
}

// SearchResult wraps the list with pagination metadata.
type SearchResult struct {
	Products []ProductListItem `json:"products"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// Suggestion is a lightweight result for typeahead/autocomplete.
type Suggestion struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Price     int       `json:"price"`
	SalePrice *int      `json:"sale_price,omitempty"`
	ImageURL  *string   `json:"image_url,omitempty"`
}
