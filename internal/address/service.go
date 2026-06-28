// service.go — saved address business logic: validation and normalisation.
package address

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Service interface {
	List(ctx context.Context, userID string) ([]Address, error)
	Create(ctx context.Context, userID string, req AddressRequest) (*Address, error)
	Update(ctx context.Context, userID, id string, req AddressRequest) (*Address, error)
	Delete(ctx context.Context, userID, id string) error
	SetDefault(ctx context.Context, userID, id string) error
}

type service struct {
	repo     Repository
	validate *validator.Validate
}

func NewService(repo Repository) Service {
	return &service{repo: repo, validate: validator.New()}
}

func (s *service) normalize(req *AddressRequest) {
	req.RecipientName = strings.TrimSpace(req.RecipientName)
	req.Line1 = strings.TrimSpace(req.Line1)
	req.Line2 = strings.TrimSpace(req.Line2)
	req.City = strings.TrimSpace(req.City)
	req.Region = strings.TrimSpace(req.Region)
	req.PostalCode = strings.TrimSpace(req.PostalCode)
	req.Country = strings.ToUpper(strings.TrimSpace(req.Country))
	req.Label = strings.TrimSpace(req.Label)
}

func (s *service) List(ctx context.Context, userID string) ([]Address, error) {
	return s.repo.List(ctx, userID)
}

func (s *service) Create(ctx context.Context, userID string, req AddressRequest) (*Address, error) {
	s.normalize(&req)
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, userID, req)
}

func (s *service) Update(ctx context.Context, userID, id string, req AddressRequest) (*Address, error) {
	s.normalize(&req)
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, userID, id, req)
}

func (s *service) Delete(ctx context.Context, userID, id string) error {
	return s.repo.Delete(ctx, userID, id)
}

func (s *service) SetDefault(ctx context.Context, userID, id string) error {
	return s.repo.SetDefault(ctx, userID, id)
}
