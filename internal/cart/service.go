package cart

import (
	"context"
	"errors"
	"fmt"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/activity"
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
	MergeStatus(ctx context.Context, userID uuid.UUID, guestSessionID *uuid.UUID) (*MergeStatusResponse, error)
	Merge(ctx context.Context, userID uuid.UUID, guestSessionID uuid.UUID, strategy string) (*CartResponse, error)
}

type service struct {
	repo     Repository
	products product.Repository
	activity activity.Logger
	validate *validator.Validate
}

func NewService(repo Repository, products product.Repository, logger activity.Logger) Service {
	if logger == nil {
		logger = activity.NoopLogger{}
	}
	return &service{
		repo:     repo,
		products: products,
		activity: logger,
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
	s.activity.LogAddToCart(ctx, userID, guestSessionID, &productID, p.CategoryID)
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

// MergeStatus reports whether the logged-in user's cart and their guest cart
// both have items and require a manual choice. Trivial cases (guest cart
// missing/empty, or user cart missing/empty) are auto-resolved and the caller
// is expected to clear the guest cookie when AutoMerged is true.
func (s *service) MergeStatus(ctx context.Context, userID uuid.UUID, guestSessionID *uuid.UUID) (*MergeStatusResponse, error) {
	if guestSessionID == nil {
		return &MergeStatusResponse{Conflict: false}, nil
	}

	guestCart, guestItems, err := s.loadCartAndItems(ctx, guestCartLookup(*guestSessionID, s.repo))
	if err != nil {
		return nil, err
	}
	if guestCart == nil || len(guestItems) == 0 {
		return &MergeStatusResponse{Conflict: false}, nil
	}

	userCart, userItems, err := s.loadCartAndItems(ctx, userCartLookup(userID, s.repo))
	if err != nil {
		return nil, err
	}

	// User has no existing cart items — fold guest into user silently.
	if userCart == nil || len(userItems) == 0 {
		merged, err := s.transferItems(ctx, userID, guestCart.ID, guestItems)
		if err != nil {
			return nil, err
		}
		return &MergeStatusResponse{Conflict: false, AutoMerged: true, UserCart: merged}, nil
	}

	// Both sides have items — surface both so the caller can prompt.
	return &MergeStatusResponse{
		Conflict:  true,
		GuestCart: toResponse(guestCart.ID, guestItems),
		UserCart:  toResponse(userCart.ID, userItems),
	}, nil
}

// Merge applies the chosen strategy: "guest" folds guest items into the user
// cart (capped at stock), "user" discards the guest cart. In both cases the
// guest cart row is deleted; the caller is expected to clear the cookie.
func (s *service) Merge(ctx context.Context, userID uuid.UUID, guestSessionID uuid.UUID, strategy string) (*CartResponse, error) {
	if strategy != MergeStrategyGuest && strategy != MergeStrategyUser {
		return nil, ErrInvalidStrategy
	}

	guestCart, guestItems, err := s.loadCartAndItems(ctx, guestCartLookup(guestSessionID, s.repo))
	if err != nil {
		return nil, err
	}

	if strategy == MergeStrategyGuest && guestCart != nil && len(guestItems) > 0 {
		return s.transferItems(ctx, userID, guestCart.ID, guestItems)
	}

	// "user" strategy, or nothing to transfer — just discard the guest cart.
	if guestCart != nil {
		if err := s.repo.DeleteCart(ctx, guestCart.ID); err != nil {
			return nil, err
		}
	}
	userCart, err := s.repo.GetOrCreateByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.buildResponse(ctx, userCart)
}

// transferItems copies guest items into the user cart with stock capping,
// deletes the guest cart, and returns the resolved user cart.
func (s *service) transferItems(ctx context.Context, userID, guestCartID uuid.UUID, guestItems []CartItemDetail) (*CartResponse, error) {
	userCart, err := s.repo.GetOrCreateByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetItems(ctx, userCart.ID)
	if err != nil {
		return nil, err
	}
	existingQty := make(map[uuid.UUID]int, len(existing))
	for _, it := range existing {
		existingQty[it.ProductID] = it.Quantity
	}

	for _, gi := range guestItems {
		room := gi.Stock - existingQty[gi.ProductID]
		if room <= 0 {
			continue
		}
		addQty := gi.Quantity
		if addQty > room {
			addQty = room
		}
		if _, err := s.repo.AddItem(ctx, userCart.ID, gi.ProductID, addQty); err != nil {
			return nil, err
		}
	}

	if err := s.repo.DeleteCart(ctx, guestCartID); err != nil {
		return nil, err
	}
	return s.buildResponse(ctx, userCart)
}

// cartLookup wraps the two ways to locate an existing cart without creating one.
type cartLookup func(ctx context.Context) (*Cart, error)

func guestCartLookup(sessionID uuid.UUID, repo Repository) cartLookup {
	return func(ctx context.Context) (*Cart, error) { return repo.GetByGuest(ctx, sessionID) }
}

func userCartLookup(userID uuid.UUID, repo Repository) cartLookup {
	return func(ctx context.Context) (*Cart, error) { return repo.GetByUser(ctx, userID) }
}

// loadCartAndItems returns (nil, nil, nil) when the cart doesn't exist.
func (s *service) loadCartAndItems(ctx context.Context, lookup cartLookup) (*Cart, []CartItemDetail, error) {
	c, err := lookup(ctx)
	if err != nil {
		if errors.Is(err, ErrCartNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	items, err := s.repo.GetItems(ctx, c.ID)
	if err != nil {
		return nil, nil, err
	}
	return c, items, nil
}

func toResponse(cartID uuid.UUID, items []CartItemDetail) *CartResponse {
	var total int64
	for _, it := range items {
		total += it.ProductPrice * int64(it.Quantity)
	}
	return &CartResponse{ID: cartID, Items: items, Total: total}
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
