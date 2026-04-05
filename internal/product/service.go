package product

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Service interface {
	Search(ctx context.Context, params SearchParams) (*SearchResult, error)
	Suggest(ctx context.Context, query string) ([]Suggestion, error)
	GetByID(ctx context.Context, id string) (*ProductDetail, error)
	Create(ctx context.Context, req CreateProductRequest) (*Product, error)
	Update(ctx context.Context, id string, req UpdateProductRequest) (*Product, error)
	Delete(ctx context.Context, id string) error
	AddImage(ctx context.Context, productID string, req AddImageRequest) (*ProductImage, error)
	DeleteImage(ctx context.Context, productID string, imageID string) error
}

type service struct {
	repo     Repository
	validate *validator.Validate
}

func NewService(repo Repository) Service {
	return &service{repo: repo, validate: validator.New()}
}

func (s *service) Search(ctx context.Context, params SearchParams) (*SearchResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	return s.repo.Search(ctx, params)
}

func (s *service) Suggest(ctx context.Context, query string) ([]Suggestion, error) {
	if query == "" {
		return []Suggestion{}, nil
	}
	return s.repo.Suggest(ctx, query, 8)
}

func (s *service) GetByID(ctx context.Context, id string) (*ProductDetail, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req CreateProductRequest) (*Product, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	p := Product{
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
		WeightG:       req.WeightG,
		DimensionsCm:  req.DimensionsCm,
	}

	if req.CategoryID != nil {
		parsed, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("invalid category_id: %w", err)
		}
		p.CategoryID = &parsed
	}
	if req.BrandID != nil {
		parsed, err := uuid.Parse(*req.BrandID)
		if err != nil {
			return nil, fmt.Errorf("invalid brand_id: %w", err)
		}
		p.BrandID = &parsed
	}

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	return created, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateProductRequest) (*Product, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) AddImage(ctx context.Context, productID string, req AddImageRequest) (*ProductImage, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}
	return s.repo.AddImage(ctx, productID, req.URL, req.IsPrimary)
}

func (s *service) DeleteImage(ctx context.Context, productID string, imageID string) error {
	return s.repo.DeleteImage(ctx, productID, imageID)
}
