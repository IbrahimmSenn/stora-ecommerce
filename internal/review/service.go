// service.go — review business logic: purchase gating, validation, moderation.
package review

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, userID, productID string, req CreateReviewRequest) (*Review, error)
	List(ctx context.Context, params ListParams) (*ListResult, error)
	Eligibility(ctx context.Context, userID, productID string) (*Eligibility, error)
	Vote(ctx context.Context, reviewID, userID string, helpful bool) error

	ListForModeration(ctx context.Context, status string, page, pageSize int) ([]ModerationItem, int, error)
	SetStatus(ctx context.Context, reviewID, status string) error
	Delete(ctx context.Context, reviewID string) error
}

type service struct {
	repo     Repository
	validate *validator.Validate
}

func NewService(repo Repository) Service {
	return &service{repo: repo, validate: validator.New()}
}

func (s *service) Create(ctx context.Context, userID, productID string, req CreateReviewRequest) (*Review, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	pid, err := uuid.Parse(productID)
	if err != nil {
		return nil, ErrProductNotFound
	}

	purchased, err := s.repo.HasPurchased(ctx, uid, pid)
	if err != nil {
		return nil, err
	}
	if !purchased {
		return nil, ErrNotPurchased
	}

	return s.repo.Create(ctx, uid, pid, req.Rating, req.Comment)
}

func (s *service) List(ctx context.Context, params ListParams) (*ListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 10
	}
	switch params.Sort {
	case SortNewest, SortHighest, SortLowest, SortHelpful:
	default:
		params.Sort = SortHelpful
	}
	if _, err := uuid.Parse(params.ProductID); err != nil {
		return nil, ErrProductNotFound
	}
	return s.repo.ListByProduct(ctx, params)
}

func (s *service) Eligibility(ctx context.Context, userID, productID string) (*Eligibility, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	pid, err := uuid.Parse(productID)
	if err != nil {
		return nil, ErrProductNotFound
	}

	purchased, err := s.repo.HasPurchased(ctx, uid, pid)
	if err != nil {
		return nil, err
	}

	e := &Eligibility{HasPurchased: purchased}
	existing, err := s.repo.GetUserReview(ctx, uid, pid)
	switch {
	case err == nil:
		e.AlreadyReviewed = true
		e.ExistingRating = &existing.Rating
		e.ExistingPending = existing.Status == StatusPending
	case err == ErrReviewNotFound:
		// no prior review — fine
	default:
		return nil, err
	}

	e.CanReview = purchased && !e.AlreadyReviewed
	return e, nil
}

func (s *service) Vote(ctx context.Context, reviewID, userID string, helpful bool) error {
	rid, err := uuid.Parse(reviewID)
	if err != nil {
		return ErrReviewNotFound
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	if helpful {
		return s.repo.AddVote(ctx, rid, uid)
	}
	return s.repo.RemoveVote(ctx, rid, uid)
}

func (s *service) ListForModeration(ctx context.Context, status string, page, pageSize int) ([]ModerationItem, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	switch status {
	case StatusPending, StatusApproved, StatusHidden, "":
	default:
		return nil, 0, fmt.Errorf("invalid status filter")
	}
	return s.repo.ListForModeration(ctx, status, page, pageSize)
}

func (s *service) SetStatus(ctx context.Context, reviewID, status string) error {
	switch status {
	case StatusPending, StatusApproved, StatusHidden:
	default:
		return fmt.Errorf("invalid status")
	}
	rid, err := uuid.Parse(reviewID)
	if err != nil {
		return ErrReviewNotFound
	}
	return s.repo.UpdateStatus(ctx, rid, status)
}

func (s *service) Delete(ctx context.Context, reviewID string) error {
	rid, err := uuid.Parse(reviewID)
	if err != nil {
		return ErrReviewNotFound
	}
	return s.repo.Delete(ctx, rid)
}
