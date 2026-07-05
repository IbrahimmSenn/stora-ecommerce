package cart

import (
	"context"
	"testing"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/product"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeStatus_NoGuestCookie(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, &noopProductRepo{}, nil)

	status, err := svc.MergeStatus(context.Background(), uuid.New(), nil)
	require.NoError(t, err)
	assert.False(t, status.Conflict)
	assert.False(t, status.AutoMerged)
}

func TestMergeStatus_EmptyGuestCart(t *testing.T) {
	repo := newStubRepo()
	guestID := uuid.New()
	repo.seedCart(Cart{ID: uuid.New(), GuestSessionID: &guestID})

	svc := NewService(repo, &noopProductRepo{}, nil)
	status, err := svc.MergeStatus(context.Background(), uuid.New(), &guestID)
	require.NoError(t, err)
	assert.False(t, status.Conflict)
	assert.False(t, status.AutoMerged)
}

func TestMergeStatus_AutoMergesWhenUserCartEmpty(t *testing.T) {
	repo := newStubRepo()
	guestID := uuid.New()
	guestCartID := uuid.New()
	productID := uuid.New()

	repo.seedCart(Cart{ID: guestCartID, GuestSessionID: &guestID})
	repo.seedItem(guestCartID, CartItemDetail{
		ID: uuid.New(), ProductID: productID, ProductPrice: 500, Quantity: 2, Stock: 10,
	})

	svc := NewService(repo, &noopProductRepo{}, nil)
	userID := uuid.New()

	status, err := svc.MergeStatus(context.Background(), userID, &guestID)
	require.NoError(t, err)
	assert.False(t, status.Conflict)
	assert.True(t, status.AutoMerged)
	require.NotNil(t, status.UserCart)
	assert.Len(t, status.UserCart.Items, 1)
	assert.Equal(t, int64(1000), status.UserCart.Total)

	_, err = repo.GetByGuest(context.Background(), guestID)
	assert.ErrorIs(t, err, ErrCartNotFound, "guest cart should be deleted after auto-merge")
}

func TestMergeStatus_ConflictExposesBothCarts(t *testing.T) {
	repo := newStubRepo()
	guestID := uuid.New()
	guestCartID := uuid.New()
	userID := uuid.New()
	userCartID := uuid.New()
	productA := uuid.New()
	productB := uuid.New()

	repo.seedCart(Cart{ID: guestCartID, GuestSessionID: &guestID})
	repo.seedItem(guestCartID, CartItemDetail{ID: uuid.New(), ProductID: productA, ProductPrice: 100, Quantity: 1, Stock: 10})

	repo.seedCart(Cart{ID: userCartID, UserID: &userID})
	repo.seedItem(userCartID, CartItemDetail{ID: uuid.New(), ProductID: productB, ProductPrice: 300, Quantity: 2, Stock: 10})

	svc := NewService(repo, &noopProductRepo{}, nil)
	status, err := svc.MergeStatus(context.Background(), userID, &guestID)
	require.NoError(t, err)
	assert.True(t, status.Conflict)
	assert.False(t, status.AutoMerged)
	require.NotNil(t, status.GuestCart)
	require.NotNil(t, status.UserCart)
	assert.Equal(t, int64(100), status.GuestCart.Total)
	assert.Equal(t, int64(600), status.UserCart.Total)
}

func TestMerge_GuestStrategy_DisjointItems(t *testing.T) {
	repo := newStubRepo()
	guestID := uuid.New()
	guestCartID := uuid.New()
	userID := uuid.New()
	userCartID := uuid.New()
	productA := uuid.New()
	productB := uuid.New()

	repo.seedCart(Cart{ID: guestCartID, GuestSessionID: &guestID})
	repo.seedItem(guestCartID, CartItemDetail{ID: uuid.New(), ProductID: productA, ProductPrice: 100, Quantity: 1, Stock: 10})
	repo.seedCart(Cart{ID: userCartID, UserID: &userID})
	repo.seedItem(userCartID, CartItemDetail{ID: uuid.New(), ProductID: productB, ProductPrice: 300, Quantity: 2, Stock: 10})

	svc := NewService(repo, &noopProductRepo{}, nil)
	resp, err := svc.Merge(context.Background(), userID, guestID, MergeStrategyGuest)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2, "both products should survive the merge")
	assert.Equal(t, int64(100+600), resp.Total)

	_, err = repo.GetByGuest(context.Background(), guestID)
	assert.ErrorIs(t, err, ErrCartNotFound)
}

func TestMerge_GuestStrategy_SumsOverlappingAndCapsAtStock(t *testing.T) {
	repo := newStubRepo()
	guestID := uuid.New()
	guestCartID := uuid.New()
	userID := uuid.New()
	userCartID := uuid.New()
	productID := uuid.New()

	// Guest wants 5, user already has 3, stock is 6 → final should cap at 6.
	repo.seedCart(Cart{ID: guestCartID, GuestSessionID: &guestID})
	repo.seedItem(guestCartID, CartItemDetail{ID: uuid.New(), ProductID: productID, ProductPrice: 1000, Quantity: 5, Stock: 6})
	repo.seedCart(Cart{ID: userCartID, UserID: &userID})
	repo.seedItem(userCartID, CartItemDetail{ID: uuid.New(), ProductID: productID, ProductPrice: 1000, Quantity: 3, Stock: 6})

	svc := NewService(repo, &noopProductRepo{}, nil)
	resp, err := svc.Merge(context.Background(), userID, guestID, MergeStrategyGuest)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, 6, resp.Items[0].Quantity, "quantity should cap at stock")
	assert.Equal(t, int64(6000), resp.Total)
}

func TestMerge_UserStrategy_DiscardsGuestCart(t *testing.T) {
	repo := newStubRepo()
	guestID := uuid.New()
	guestCartID := uuid.New()
	userID := uuid.New()
	userCartID := uuid.New()
	productA := uuid.New()
	productB := uuid.New()

	repo.seedCart(Cart{ID: guestCartID, GuestSessionID: &guestID})
	repo.seedItem(guestCartID, CartItemDetail{ID: uuid.New(), ProductID: productA, ProductPrice: 100, Quantity: 1, Stock: 10})
	repo.seedCart(Cart{ID: userCartID, UserID: &userID})
	repo.seedItem(userCartID, CartItemDetail{ID: uuid.New(), ProductID: productB, ProductPrice: 300, Quantity: 2, Stock: 10})

	svc := NewService(repo, &noopProductRepo{}, nil)
	resp, err := svc.Merge(context.Background(), userID, guestID, MergeStrategyUser)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1, "user cart should be untouched")
	assert.Equal(t, productB, resp.Items[0].ProductID)

	_, err = repo.GetByGuest(context.Background(), guestID)
	assert.ErrorIs(t, err, ErrCartNotFound, "guest cart should be discarded")
}

func TestMerge_InvalidStrategy(t *testing.T) {
	repo := newStubRepo()
	svc := NewService(repo, &noopProductRepo{}, nil)

	_, err := svc.Merge(context.Background(), uuid.New(), uuid.New(), "nope")
	assert.ErrorIs(t, err, ErrInvalidStrategy)
}

// --- stubs ---

type stubRepo struct {
	carts    map[uuid.UUID]*Cart
	items    map[uuid.UUID][]CartItemDetail
	products map[uuid.UUID]stubProduct
}

type stubProduct struct {
	price int64
	stock int
}

func newStubRepo() *stubRepo {
	return &stubRepo{
		carts:    map[uuid.UUID]*Cart{},
		items:    map[uuid.UUID][]CartItemDetail{},
		products: map[uuid.UUID]stubProduct{},
	}
}

func (s *stubRepo) seedCart(c Cart) {
	s.carts[c.ID] = &c
}

// seedItem registers the line in the cart AND records the product's price/stock
// so later AddItem calls against the same product resolve consistently.
func (s *stubRepo) seedItem(cartID uuid.UUID, item CartItemDetail) {
	s.items[cartID] = append(s.items[cartID], item)
	s.products[item.ProductID] = stubProduct{price: item.ProductPrice, stock: item.Stock}
}

func (s *stubRepo) GetOrCreateByUser(_ context.Context, userID uuid.UUID) (*Cart, error) {
	for _, c := range s.carts {
		if c.UserID != nil && *c.UserID == userID {
			return c, nil
		}
	}
	c := Cart{ID: uuid.New(), UserID: &userID}
	s.carts[c.ID] = &c
	return &c, nil
}

func (s *stubRepo) GetOrCreateByGuest(_ context.Context, sessionID uuid.UUID) (*Cart, error) {
	for _, c := range s.carts {
		if c.GuestSessionID != nil && *c.GuestSessionID == sessionID {
			return c, nil
		}
	}
	c := Cart{ID: uuid.New(), GuestSessionID: &sessionID}
	s.carts[c.ID] = &c
	return &c, nil
}

func (s *stubRepo) GetByUser(_ context.Context, userID uuid.UUID) (*Cart, error) {
	for _, c := range s.carts {
		if c.UserID != nil && *c.UserID == userID {
			return c, nil
		}
	}
	return nil, ErrCartNotFound
}

func (s *stubRepo) GetByGuest(_ context.Context, sessionID uuid.UUID) (*Cart, error) {
	for _, c := range s.carts {
		if c.GuestSessionID != nil && *c.GuestSessionID == sessionID {
			return c, nil
		}
	}
	return nil, ErrCartNotFound
}

func (s *stubRepo) GetByID(_ context.Context, cartID uuid.UUID) (*Cart, error) {
	if c, ok := s.carts[cartID]; ok {
		return c, nil
	}
	return nil, ErrCartNotFound
}

func (s *stubRepo) AddItem(_ context.Context, cartID, productID uuid.UUID, quantity int) (*CartItem, error) {
	for i, it := range s.items[cartID] {
		if it.ProductID == productID {
			s.items[cartID][i].Quantity += quantity
			return &CartItem{ID: it.ID, CartID: cartID, ProductID: productID, Quantity: s.items[cartID][i].Quantity}, nil
		}
	}
	p := s.products[productID]
	item := CartItemDetail{ID: uuid.New(), ProductID: productID, Quantity: quantity, ProductPrice: p.price, Stock: p.stock}
	s.items[cartID] = append(s.items[cartID], item)
	return &CartItem{ID: item.ID, CartID: cartID, ProductID: productID, Quantity: quantity}, nil
}

func (s *stubRepo) UpdateItemQuantity(_ context.Context, cartID, productID uuid.UUID, quantity int) (*CartItem, error) {
	for i, it := range s.items[cartID] {
		if it.ProductID == productID {
			s.items[cartID][i].Quantity = quantity
			return &CartItem{ID: it.ID, CartID: cartID, ProductID: productID, Quantity: quantity}, nil
		}
	}
	return nil, ErrItemNotFound
}

func (s *stubRepo) RemoveItem(_ context.Context, cartID, productID uuid.UUID) error {
	list := s.items[cartID]
	for i, it := range list {
		if it.ProductID == productID {
			s.items[cartID] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return ErrItemNotFound
}

func (s *stubRepo) ClearCart(_ context.Context, cartID uuid.UUID) error {
	s.items[cartID] = nil
	return nil
}

func (s *stubRepo) DeleteCart(_ context.Context, cartID uuid.UUID) error {
	delete(s.carts, cartID)
	delete(s.items, cartID)
	return nil
}

func (s *stubRepo) GetItems(_ context.Context, cartID uuid.UUID) ([]CartItemDetail, error) {
	out := make([]CartItemDetail, len(s.items[cartID]))
	copy(out, s.items[cartID])
	if out == nil {
		out = []CartItemDetail{}
	}
	return out, nil
}

type noopProductRepo struct{}

func (noopProductRepo) Search(context.Context, product.SearchParams) (*product.SearchResult, error) {
	return nil, nil
}
func (noopProductRepo) Suggest(context.Context, string, int) ([]product.Suggestion, error) {
	return nil, nil
}
func (noopProductRepo) GetByID(context.Context, string) (*product.ProductDetail, error) {
	return nil, nil
}
func (noopProductRepo) Create(context.Context, product.Product) (*product.Product, error) {
	return nil, nil
}
func (noopProductRepo) Update(context.Context, string, product.UpdateProductRequest) (*product.Product, error) {
	return nil, nil
}
func (noopProductRepo) Delete(context.Context, string) error { return nil }
func (noopProductRepo) AddImage(context.Context, string, string, bool) (*product.ProductImage, error) {
	return nil, nil
}
func (noopProductRepo) AddImageWithVariants(context.Context, string, string, string, string, string, bool) (*product.ProductImage, error) {
	return nil, nil
}
func (noopProductRepo) DeleteImage(context.Context, string, string) error { return nil }
func (noopProductRepo) GetImages(context.Context, string) ([]product.ProductImage, error) {
	return nil, nil
}
func (noopProductRepo) ListByIDs(context.Context, []string) ([]product.ProductListItem, error) {
	return nil, nil
}
func (noopProductRepo) Candidates(context.Context, []string, int) ([]product.Candidate, error) {
	return nil, nil
}
func (noopProductRepo) CategoryBrandFor(context.Context, []string) (map[uuid.UUID]product.CategoryBrand, error) {
	return nil, nil
}
