package cart

import (
	"context"
	"fmt"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/product"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Service interface {
	GetCart(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID) (*CartResponse, error)
	AddItem(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID, req AddItemRequest) (*CartResponse, error)
	UpdateItem(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID, req UpdateItemRequest) (*CartResponse, error)
	RemoveItem(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID, productID string) (*CartResponse, error)
	ClearCart(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID) error
}

type service struct {
	repo     Repository
	products product.Repository
	validate *validator.Validate
}

func NewService(repo Repository, products product.Repository) Service {
	return &service{
		repo:     repo,
		products: products,
		validate: validator.New(),
	}
}

func (s *service) GetCart(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID) (*CartResponse, error) {
	c, err := s.resolveCart(ctx, userID, guestSessionID)
	if err != nil {
		return nil, err
	}
	return s.buildResponse(ctx, c)
}

func (s *service) AddItem(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID, req AddItemRequest) (*CartResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("invalid product_id: %w", err)
	}

	p, err := s.products.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("lookup product: %w", err)
	}
	if p.StockQuantity < req.Quantity {
		return nil, ErrOutOfStock
	}

	c, err := s.resolveCart(ctx, userID, guestSessionID)
	if err != nil {
		return nil, err
	}

	// Check combined quantity if item already exists in cart.
	items, err := s.repo.GetItems(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ProductID == productID {
			if item.Quantity+req.Quantity > p.StockQuantity {
				return nil, ErrOutOfStock
			}
			break
		}
	}

	if _, err := s.repo.AddItem(ctx, c.ID, productID, req.Quantity); err != nil {
		return nil, err
	}
	return s.buildResponse(ctx, c)
}

func (s *service) UpdateItem(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID, req UpdateItemRequest) (*CartResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("invalid product_id: %w", err)
	}

	p, err := s.products.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("lookup product: %w", err)
	}
	if p.StockQuantity < req.Quantity {
		return nil, ErrOutOfStock
	}

	c, err := s.resolveCart(ctx, userID, guestSessionID)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.UpdateItemQuantity(ctx, c.ID, productID, req.Quantity); err != nil {
		return nil, err
	}
	return s.buildResponse(ctx, c)
}

func (s *service) RemoveItem(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID, productID string) (*CartResponse, error) {
	pid, err := uuid.Parse(productID)
	if err != nil {
		return nil, fmt.Errorf("invalid product_id: %w", err)
	}

	c, err := s.resolveCart(ctx, userID, guestSessionID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.RemoveItem(ctx, c.ID, pid); err != nil {
		return nil, err
	}
	return s.buildResponse(ctx, c)
}

func (s *service) ClearCart(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID) error {
	c, err := s.resolveCart(ctx, userID, guestSessionID)
	if err != nil {
		return err
	}
	return s.repo.ClearCart(ctx, c.ID)
}

// resolveCart finds or creates the cart for the given user or guest session.
func (s *service) resolveCart(ctx context.Context, userID *uuid.UUID, guestSessionID *uuid.UUID) (*Cart, error) {
	if userID != nil {
		return s.repo.GetOrCreateByUser(ctx, *userID)
	}
	if guestSessionID != nil {
		return s.repo.GetOrCreateByGuest(ctx, *guestSessionID)
	}
	return nil, ErrNoOwner
}

// buildResponse loads items and computes the total.
func (s *service) buildResponse(ctx context.Context, c *Cart) (*CartResponse, error) {
	items, err := s.repo.GetItems(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	var total int64
	for _, item := range items {
		total += item.ProductPrice * int64(item.Quantity)
	}

	return &CartResponse{
		ID:    c.ID,
		Items: items,
		Total: total,
	}, nil
}
