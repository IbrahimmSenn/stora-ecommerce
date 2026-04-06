// service.go — brand business logic and input validation.
package brand

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
)

type Service interface {
	List(ctx context.Context) ([]Brand, error)
	GetByID(ctx context.Context, id string) (*Brand, error)
	Create(ctx context.Context, req CreateBrandRequest) (*Brand, error)
}

type service struct {
	repo     Repository
	validate *validator.Validate
}

func NewService(repo Repository) Service {
	return &service{repo: repo, validate: validator.New()}
}

func (s *service) List(ctx context.Context) ([]Brand, error) {
	return s.repo.List(ctx)
}

func (s *service) GetByID(ctx context.Context, id string) (*Brand, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req CreateBrandRequest) (*Brand, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}
	b, err := s.repo.Create(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("create brand: %w", err)
	}
	return b, nil
}
