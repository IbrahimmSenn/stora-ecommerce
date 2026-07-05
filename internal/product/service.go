// service.go — product validation, pagination clamping, and UUID parsing.
package product

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/imageproc"
)

type Service interface {
	Search(ctx context.Context, params SearchParams) (*SearchResult, error)
	Suggest(ctx context.Context, query string) ([]Suggestion, error)
	GetByID(ctx context.Context, id string) (*ProductDetail, error)
	Create(ctx context.Context, req CreateProductRequest) (*Product, error)
	BulkCreate(ctx context.Context, reqs []CreateProductRequest) *BulkResult
	Update(ctx context.Context, id string, req UpdateProductRequest) (*Product, error)
	Delete(ctx context.Context, id string) error
	AddImage(ctx context.Context, productID string, req AddImageRequest) (*ProductImage, error)
	UploadImage(ctx context.Context, productID string, src io.Reader, isPrimary bool) (*ProductImage, error)
	DeleteImage(ctx context.Context, productID string, imageID string) error
}

// ImageProcessor decodes an uploaded image and writes the sized variants,
// returning their public URLs. Satisfied by *imageproc.Processor.
type ImageProcessor interface {
	Process(id string, src io.Reader) (*imageproc.Variants, error)
}

type service struct {
	repo     Repository
	validate *validator.Validate
	images   ImageProcessor
}

type Option func(*service)

// WithImageProcessor enables the upload pipeline. Without it, UploadImage
// returns ErrUploadsDisabled.
func WithImageProcessor(p ImageProcessor) Option {
	return func(s *service) { s.images = p }
}

func NewService(repo Repository, opts ...Option) Service {
	s := &service{repo: repo, validate: validator.New()}
	for _, opt := range opts {
		opt(s)
	}
	return s
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
	if req.SalePrice != nil && *req.SalePrice >= req.Price {
		return nil, ErrInvalidSalePrice
	}

	p := Product{
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		SalePrice:     req.SalePrice,
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

// BulkCreate inserts many products, collecting per-row failures rather than
// aborting the whole batch. Each row goes through the same validation as a
// single Create. Used by the JSON and CSV admin bulk-upload paths.
func (s *service) BulkCreate(ctx context.Context, reqs []CreateProductRequest) *BulkResult {
	res := &BulkResult{Errors: []BulkItemError{}}
	for i, req := range reqs {
		if _, err := s.Create(ctx, req); err != nil {
			res.Failed++
			msg := "could not create product"
			var ve validator.ValidationErrors
			if errors.As(err, &ve) {
				msg = formatValidationErrors(ve)
			} else if err != nil {
				msg = err.Error()
			}
			res.Errors = append(res.Errors, BulkItemError{Index: i, Name: req.Name, Error: msg})
			continue
		}
		res.Created++
	}
	return res
}

func (s *service) Update(ctx context.Context, id string, req UpdateProductRequest) (*Product, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}
	// When both the new sale price and new price arrive together we can reject
	// early with a friendly message. If only one is present, the DB check
	// constraint is the backstop (mapped to ErrInvalidSalePrice in the repo).
	if req.SalePrice != nil && req.Price != nil && *req.SalePrice >= *req.Price {
		return nil, ErrInvalidSalePrice
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

// UploadImage runs the file through the image pipeline (generating thumbnail/
// card/full variants) and records a product_images row pointing at them. The
// canonical `url` is the full-size variant so legacy readers still get an image.
func (s *service) UploadImage(ctx context.Context, productID string, src io.Reader, isPrimary bool) (*ProductImage, error) {
	if s.images == nil {
		return nil, ErrUploadsDisabled
	}
	if _, err := uuid.Parse(productID); err != nil {
		return nil, ErrProductNotFound
	}

	token := uuid.NewString()
	v, err := s.images.Process(token, src)
	if err != nil {
		return nil, err
	}
	return s.repo.AddImageWithVariants(ctx, productID, v.FullURL, v.ThumbnailURL, v.CardURL, v.FullURL, isPrimary)
}

func (s *service) DeleteImage(ctx context.Context, productID string, imageID string) error {
	return s.repo.DeleteImage(ctx, productID, imageID)
}
