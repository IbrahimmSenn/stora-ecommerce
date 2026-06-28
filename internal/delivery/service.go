// service.go — delivery option business logic.
package delivery

import (
	"context"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// codePattern whitelists option codes: lowercase letters, digits, hyphens. The
// code is stored on orders, so keep it stable and URL/identifier-safe.
var codePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

type Service interface {
	List(ctx context.Context, activeOnly bool) ([]DeliveryOption, error)
	Create(ctx context.Context, req CreateRequest) (*DeliveryOption, error)
	Update(ctx context.Context, id string, req UpdateRequest) (*DeliveryOption, error)
	Delete(ctx context.Context, id string) error
	// Rate resolves a method code to its cost. ok=false for unknown/inactive
	// codes. Implements the orders.ShippingRater contract.
	Rate(ctx context.Context, code string) (cents int64, ok bool, err error)
}

type service struct {
	repo     Repository
	validate *validator.Validate
}

func NewService(repo Repository) Service {
	return &service{repo: repo, validate: validator.New()}
}

func (s *service) List(ctx context.Context, activeOnly bool) ([]DeliveryOption, error) {
	return s.repo.List(ctx, activeOnly)
}

func (s *service) Create(ctx context.Context, req CreateRequest) (*DeliveryOption, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}
	code := strings.ToLower(strings.TrimSpace(req.Code))
	if !codePattern.MatchString(code) {
		return nil, ErrInvalidCode
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	return s.repo.Create(ctx, DeliveryOption{
		Code:       code,
		Label:      strings.TrimSpace(req.Label),
		PriceCents: req.PriceCents,
		EtaLabel:   strings.TrimSpace(req.EtaLabel),
		SortOrder:  req.SortOrder,
		Active:     active,
	})
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (*DeliveryOption, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	return s.repo.Update(ctx, id, DeliveryOption{
		Label:      strings.TrimSpace(req.Label),
		PriceCents: req.PriceCents,
		EtaLabel:   strings.TrimSpace(req.EtaLabel),
		SortOrder:  req.SortOrder,
		Active:     active,
	})
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) Rate(ctx context.Context, code string) (int64, bool, error) {
	return s.repo.RateByCode(ctx, code)
}
