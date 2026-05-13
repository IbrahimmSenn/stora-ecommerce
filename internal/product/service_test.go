package product

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Search_DefaultPagination(t *testing.T) {
	svc := NewService(&stubRepo{})

	result, err := svc.Search(context.Background(), SearchParams{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PageSize)
}

func TestService_Search_ClampPageSize(t *testing.T) {
	svc := NewService(&stubRepo{})

	result, err := svc.Search(context.Background(), SearchParams{PageSize: 500})
	require.NoError(t, err)
	assert.Equal(t, 20, result.PageSize, "page size >100 should be clamped to default")
}

func TestService_Create_Validation(t *testing.T) {
	svc := NewService(&stubRepo{})

	// Missing required name.
	_, err := svc.Create(context.Background(), CreateProductRequest{
		Price:   1000,
		WeightG: intPtr(100),
	})
	assert.Error(t, err)
}

func TestService_Create_InvalidCategoryID(t *testing.T) {
	svc := NewService(&stubRepo{})
	badID := "not-a-uuid"

	_, err := svc.Create(context.Background(), CreateProductRequest{
		Name:       "Test Product",
		Price:      1000,
		WeightG:    intPtr(100),
		CategoryID: &badID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid category_id")
}

func TestService_Create_InvalidBrandID(t *testing.T) {
	svc := NewService(&stubRepo{})
	badID := "not-a-uuid"

	_, err := svc.Create(context.Background(), CreateProductRequest{
		Name:    "Test Product",
		Price:   1000,
		WeightG: intPtr(100),
		BrandID: &badID,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid brand_id")
}

// stubRepo is a minimal implementation that returns empty results.
// It's enough to test service-layer logic (validation, pagination defaults).
type stubRepo struct{}

func (s *stubRepo) Search(_ context.Context, params SearchParams) (*SearchResult, error) {
	return &SearchResult{
		Products: []ProductListItem{},
		Total:    0,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

func (s *stubRepo) Suggest(_ context.Context, _ string, _ int) ([]Suggestion, error) {
	return []Suggestion{}, nil
}

func (s *stubRepo) GetByID(_ context.Context, _ string) (*ProductDetail, error) {
	return nil, ErrProductNotFound
}

func (s *stubRepo) Create(_ context.Context, p Product) (*Product, error) {
	return &p, nil
}

func (s *stubRepo) Update(_ context.Context, _ string, _ UpdateProductRequest) (*Product, error) {
	return nil, ErrProductNotFound
}

func (s *stubRepo) Delete(_ context.Context, _ string) error {
	return ErrProductNotFound
}

func (s *stubRepo) AddImage(_ context.Context, _ string, _ string, _ bool) (*ProductImage, error) {
	return &ProductImage{}, nil
}

func (s *stubRepo) DeleteImage(_ context.Context, _ string, _ string) error {
	return ErrImageNotFound
}

func (s *stubRepo) GetImages(_ context.Context, _ string) ([]ProductImage, error) {
	return []ProductImage{}, nil
}

func intPtr(i int) *int { return &i }
