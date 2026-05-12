package orders

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/cart"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/crypto"
)

const testHexKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func newTestService(t *testing.T, repo *stubRepo, carts cart.Service) Service {
	return newTestServiceWithRefunder(t, repo, carts, nil)
}

func newTestServiceWithRefunder(t *testing.T, repo *stubRepo, carts cart.Service, refunder Refunder) Service {
	t.Helper()
	enc, err := crypto.NewEncryptor(testHexKey)
	require.NoError(t, err)
	return NewService(repo, carts, enc, refunder)
}

type stubRefunder struct {
	calls []uuid.UUID
	err   error
}

func (s *stubRefunder) RefundOrder(_ context.Context, orderID uuid.UUID) error {
	s.calls = append(s.calls, orderID)
	return s.err
}

func validRequest() CheckoutRequest {
	return CheckoutRequest{
		Email:          "buyer@example.com",
		Phone:          "5551234567",
		ShippingMethod: ShippingStandard,
		Address: CheckoutAddressRequest{
			RecipientName: "Buyer",
			Line1:         "1 Demo St",
			City:          "Townsville",
			Region:        "TS",
			PostalCode:    "00000",
			Country:       "US",
		},
	}
}

func TestCheckout_EmptyCart(t *testing.T) {
	repo := newStubRepo()
	carts := &stubCart{cartID: uuid.New(), items: nil}
	svc := newTestService(t, repo, carts)

	uid := uuid.New()
	_, err := svc.Checkout(context.Background(), &uid, nil, validRequest())
	assert.ErrorIs(t, err, ErrCartEmpty)
	assert.Empty(t, repo.orders, "no order should be inserted when cart is empty")
}

func TestCheckout_NoOwner(t *testing.T) {
	repo := newStubRepo()
	carts := &stubCart{cartID: uuid.New()}
	svc := newTestService(t, repo, carts)

	_, err := svc.Checkout(context.Background(), nil, nil, validRequest())
	assert.ErrorIs(t, err, ErrNoOwner)
}

func TestCheckout_InvalidShippingMethod(t *testing.T) {
	repo := newStubRepo()
	carts := &stubCart{cartID: uuid.New()}
	svc := newTestService(t, repo, carts)

	req := validRequest()
	req.ShippingMethod = "drone" // not in oneof=standard express
	uid := uuid.New()
	_, err := svc.Checkout(context.Background(), &uid, nil, req)
	require.Error(t, err)
	// validator rejects before we reach the rate-map check
	assert.NotErrorIs(t, err, ErrCartEmpty)
}

func TestCheckout_StockChangedDropsBelowCart(t *testing.T) {
	productID := uuid.New()
	repo := newStubRepo()
	repo.products[productID] = LockedProduct{ID: productID, Name: "Widget", Price: 1000, Stock: 1} // stock dropped

	carts := &stubCart{
		cartID: uuid.New(),
		items: []cart.CartItemDetail{{
			ID: uuid.New(), ProductID: productID, ProductPrice: 1000, Quantity: 3, Stock: 5,
		}},
	}
	svc := newTestService(t, repo, carts)

	uid := uuid.New()
	_, err := svc.Checkout(context.Background(), &uid, nil, validRequest())
	assert.ErrorIs(t, err, ErrStockChanged)
	assert.Empty(t, repo.orders, "tx must roll back on stock conflict")
}

func TestCheckout_PriceChangedTriggersConflict(t *testing.T) {
	productID := uuid.New()
	repo := newStubRepo()
	repo.products[productID] = LockedProduct{ID: productID, Name: "Widget", Price: 999, Stock: 5} // price changed

	carts := &stubCart{
		cartID: uuid.New(),
		items: []cart.CartItemDetail{{
			ID: uuid.New(), ProductID: productID, ProductPrice: 1000, Quantity: 1, Stock: 5,
		}},
	}
	svc := newTestService(t, repo, carts)

	uid := uuid.New()
	_, err := svc.Checkout(context.Background(), &uid, nil, validRequest())
	assert.ErrorIs(t, err, ErrStockChanged)
}

func TestCheckout_SuccessCreatesEncryptedOrder(t *testing.T) {
	productID := uuid.New()
	repo := newStubRepo()
	repo.products[productID] = LockedProduct{ID: productID, Name: "Widget", Price: 1000, Stock: 5}

	cartID := uuid.New()
	carts := &stubCart{
		cartID: cartID,
		items: []cart.CartItemDetail{{
			ID: uuid.New(), ProductID: productID, ProductPrice: 1000, Quantity: 2, Stock: 5,
		}},
	}
	svc := newTestService(t, repo, carts)

	uid := uuid.New()
	resp, err := svc.Checkout(context.Background(), &uid, nil, validRequest())
	require.NoError(t, err)

	// Subtotal = 2 * 1000; standard shipping = 500; total = 2500.
	assert.Equal(t, int64(2000), resp.Order.SubtotalCents)
	assert.Equal(t, int64(500), resp.Order.ShippingCents)
	assert.Equal(t, int64(2500), resp.Order.TotalCents)
	assert.Equal(t, StatusPendingPayment, resp.Order.Status)
	assert.Equal(t, "buyer@example.com", resp.Order.Email)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, 2, resp.Items[0].Quantity)
	assert.Equal(t, "Widget", resp.Items[0].ProductName)

	// Stock decremented; cart cleared.
	assert.Equal(t, 3, repo.products[productID].Stock)
	assert.True(t, repo.clearedCarts[cartID])

	// PII stored encrypted, not as plaintext bytea.
	require.Len(t, repo.orders, 1)
	stored := repo.orders[resp.Order.ID]
	assert.NotEqual(t, []byte("buyer@example.com"), stored.EmailEnc)
	assert.NotContains(t, string(stored.EmailEnc), "buyer@example.com")
}

func TestGetByID_OwnershipMismatchUser(t *testing.T) {
	repo := newStubRepo()
	owner := uuid.New()
	id := repo.seedOrder(orderRow{UserID: &owner, Status: StatusPaid})

	carts := &stubCart{}
	svc := newTestService(t, repo, carts)

	intruder := uuid.New()
	_, err := svc.GetByID(context.Background(), &intruder, nil, id)
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestGetByID_GuestSeesOwnOrder(t *testing.T) {
	repo := newStubRepo()
	guest := uuid.New()
	id := repo.seedOrder(orderRow{GuestSessionID: &guest, Status: StatusPendingPayment})

	carts := &stubCart{}
	svc := newTestService(t, repo, carts)

	resp, err := svc.GetByID(context.Background(), nil, &guest, id)
	require.NoError(t, err)
	assert.Equal(t, id, resp.Order.ID)
}

func TestCancel_ShippedRejected(t *testing.T) {
	repo := newStubRepo()
	owner := uuid.New()
	id := repo.seedOrder(orderRow{UserID: &owner, Status: StatusShipped})

	carts := &stubCart{}
	svc := newTestService(t, repo, carts)

	_, err := svc.Cancel(context.Background(), &owner, nil, id)
	assert.ErrorIs(t, err, ErrNotCancellable)
}

func TestCancel_RestoresStockAndStatus(t *testing.T) {
	repo := newStubRepo()
	productID := uuid.New()
	repo.products[productID] = LockedProduct{ID: productID, Name: "Widget", Price: 1000, Stock: 4}

	owner := uuid.New()
	id := repo.seedOrder(orderRow{UserID: &owner, Status: StatusPendingPayment})
	pid := productID
	repo.itemsByOrder[id] = []OrderItem{{ID: uuid.New(), OrderID: id, ProductID: &pid, Quantity: 3}}

	carts := &stubCart{}
	svc := newTestService(t, repo, carts)

	resp, err := svc.Cancel(context.Background(), &owner, nil, id)
	require.NoError(t, err)
	assert.Equal(t, StatusCancelled, resp.Order.Status)
	assert.Equal(t, 7, repo.products[productID].Stock, "3 units should be restocked")
}

func TestCancel_PaidOrderRefundsAndMarksRefunded(t *testing.T) {
	repo := newStubRepo()
	productID := uuid.New()
	repo.products[productID] = LockedProduct{ID: productID, Name: "Widget", Price: 1000, Stock: 4}

	owner := uuid.New()
	id := repo.seedOrder(orderRow{UserID: &owner, Status: StatusPaid})
	pid := productID
	repo.itemsByOrder[id] = []OrderItem{{ID: uuid.New(), OrderID: id, ProductID: &pid, Quantity: 2}}

	refunder := &stubRefunder{}
	svc := newTestServiceWithRefunder(t, repo, &stubCart{}, refunder)

	resp, err := svc.Cancel(context.Background(), &owner, nil, id)
	require.NoError(t, err)
	assert.Equal(t, StatusRefunded, resp.Order.Status, "paid order ends up refunded, not cancelled")
	assert.Equal(t, 6, repo.products[productID].Stock, "stock restocked from 4 → 6")
	require.Len(t, refunder.calls, 1)
	assert.Equal(t, id, refunder.calls[0])
}

func TestCancel_PaidOrderWithoutRefunderErrors(t *testing.T) {
	repo := newStubRepo()
	owner := uuid.New()
	id := repo.seedOrder(orderRow{UserID: &owner, Status: StatusPaid})

	svc := newTestService(t, repo, &stubCart{}) // no refunder

	_, err := svc.Cancel(context.Background(), &owner, nil, id)
	assert.ErrorIs(t, err, ErrRefundUnavailable)
	// Status unchanged; never touched the DB beyond the initial load.
	assert.Equal(t, StatusPaid, repo.orders[id].Status)
}

func TestCancel_RefundFailureLeavesOrderPaid(t *testing.T) {
	repo := newStubRepo()
	productID := uuid.New()
	repo.products[productID] = LockedProduct{ID: productID, Name: "Widget", Price: 1000, Stock: 4}

	owner := uuid.New()
	id := repo.seedOrder(orderRow{UserID: &owner, Status: StatusPaid})
	pid := productID
	repo.itemsByOrder[id] = []OrderItem{{ID: uuid.New(), OrderID: id, ProductID: &pid, Quantity: 2}}

	refunder := &stubRefunder{err: errors.New("stripe boom")}
	svc := newTestServiceWithRefunder(t, repo, &stubCart{}, refunder)

	_, err := svc.Cancel(context.Background(), &owner, nil, id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stripe boom")
	// Refund failed → no restock, status stays paid.
	assert.Equal(t, StatusPaid, repo.orders[id].Status)
	assert.Equal(t, 4, repo.products[productID].Stock)
}

// --- stubs ------------------------------------------------------------------

type stubRepo struct {
	products     map[uuid.UUID]LockedProduct
	orders       map[uuid.UUID]*orderRow
	itemsByOrder map[uuid.UUID][]OrderItem
	addresses    map[uuid.UUID]*addressRow
	clearedCarts map[uuid.UUID]bool
}

func newStubRepo() *stubRepo {
	return &stubRepo{
		products:     map[uuid.UUID]LockedProduct{},
		orders:       map[uuid.UUID]*orderRow{},
		itemsByOrder: map[uuid.UUID][]OrderItem{},
		addresses:    map[uuid.UUID]*addressRow{},
		clearedCarts: map[uuid.UUID]bool{},
	}
}

func (s *stubRepo) seedOrder(row orderRow) uuid.UUID {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	if row.OrderNumber == "" {
		row.OrderNumber = "ORD-TEST"
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
		row.UpdatedAt = row.CreatedAt
	}
	c := row
	s.orders[c.ID] = &c
	s.addresses[c.ID] = &addressRow{} // empty ciphertexts decrypt to ""
	return c.ID
}

func (s *stubRepo) WithTx(_ context.Context, fn func(TxRepo) error) error {
	return fn(&stubTx{parent: s})
}

func (s *stubRepo) GetByID(_ context.Context, id uuid.UUID) (*orderRow, []OrderItem, *addressRow, error) {
	o, ok := s.orders[id]
	if !ok {
		return nil, nil, nil, ErrOrderNotFound
	}
	return o, s.itemsByOrder[id], s.addresses[id], nil
}

func (s *stubRepo) ListByUser(_ context.Context, userID uuid.UUID, _ string, _, _ *time.Time) ([]OrderSummary, error) {
	var out []OrderSummary
	for _, o := range s.orders {
		if o.UserID != nil && *o.UserID == userID {
			out = append(out, OrderSummary{ID: o.ID, OrderNumber: o.OrderNumber, Status: o.Status, TotalCents: o.TotalCents, ItemCount: len(s.itemsByOrder[o.ID]), CreatedAt: o.CreatedAt})
		}
	}
	return out, nil
}

func (s *stubRepo) ListByGuest(_ context.Context, guestID uuid.UUID, _ string, _, _ *time.Time) ([]OrderSummary, error) {
	var out []OrderSummary
	for _, o := range s.orders {
		if o.GuestSessionID != nil && *o.GuestSessionID == guestID {
			out = append(out, OrderSummary{ID: o.ID, OrderNumber: o.OrderNumber, Status: o.Status, TotalCents: o.TotalCents, ItemCount: len(s.itemsByOrder[o.ID]), CreatedAt: o.CreatedAt})
		}
	}
	return out, nil
}

func (s *stubRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) error {
	o, ok := s.orders[id]
	if !ok {
		return ErrOrderNotFound
	}
	o.Status = status
	return nil
}

func (s *stubRepo) ItemsForRestock(_ context.Context, orderID uuid.UUID) ([]OrderItem, error) {
	out := []OrderItem{}
	for _, it := range s.itemsByOrder[orderID] {
		if it.ProductID != nil {
			out = append(out, it)
		}
	}
	return out, nil
}

type stubTx struct {
	parent *stubRepo
}

func (t *stubTx) LockProductForUpdate(_ context.Context, productID uuid.UUID) (*LockedProduct, error) {
	p, ok := t.parent.products[productID]
	if !ok {
		return nil, errors.New("stub: product not found")
	}
	c := p
	return &c, nil
}

func (t *stubTx) DecrementStock(_ context.Context, productID uuid.UUID, qty int) error {
	p := t.parent.products[productID]
	p.Stock -= qty
	t.parent.products[productID] = p
	return nil
}

func (t *stubTx) IncrementStock(_ context.Context, productID uuid.UUID, qty int) error {
	p := t.parent.products[productID]
	p.Stock += qty
	t.parent.products[productID] = p
	return nil
}

func (t *stubTx) DeleteCartItems(_ context.Context, cartID uuid.UUID) error {
	t.parent.clearedCarts[cartID] = true
	return nil
}

func (t *stubTx) CreateOrder(_ context.Context, row *orderRow) error {
	row.ID = uuid.New()
	row.OrderNumber = "ORD-STUB"
	row.CreatedAt = time.Now()
	row.UpdatedAt = row.CreatedAt
	c := *row
	t.parent.orders[row.ID] = &c
	return nil
}

func (t *stubTx) CreateOrderItem(_ context.Context, item *OrderItem) error {
	item.ID = uuid.New()
	item.CreatedAt = time.Now()
	t.parent.itemsByOrder[item.OrderID] = append(t.parent.itemsByOrder[item.OrderID], *item)
	return nil
}

func (t *stubTx) CreateShippingAddress(_ context.Context, orderID uuid.UUID, addr *addressRow) error {
	c := *addr
	t.parent.addresses[orderID] = &c
	return nil
}

func (t *stubTx) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return t.parent.UpdateStatus(ctx, id, status)
}

// stubCart only implements the subset of cart.Service that the orders service actually uses.
type stubCart struct {
	cartID uuid.UUID
	items  []cart.CartItemDetail
}

func (c *stubCart) GetCart(_ context.Context, _ *uuid.UUID, _ *uuid.UUID) (*cart.CartResponse, error) {
	var total int64
	for _, it := range c.items {
		total += it.ProductPrice * int64(it.Quantity)
	}
	if c.items == nil {
		c.items = []cart.CartItemDetail{}
	}
	return &cart.CartResponse{ID: c.cartID, Items: c.items, Total: total}, nil
}
func (c *stubCart) AddItem(context.Context, *uuid.UUID, *uuid.UUID, cart.AddItemRequest) (*cart.CartResponse, error) {
	return nil, nil
}
func (c *stubCart) UpdateItem(context.Context, *uuid.UUID, *uuid.UUID, cart.UpdateItemRequest) (*cart.CartResponse, error) {
	return nil, nil
}
func (c *stubCart) RemoveItem(context.Context, *uuid.UUID, *uuid.UUID, string) (*cart.CartResponse, error) {
	return nil, nil
}
func (c *stubCart) ClearCart(context.Context, *uuid.UUID, *uuid.UUID) error { return nil }
func (c *stubCart) MergeStatus(context.Context, uuid.UUID, *uuid.UUID) (*cart.MergeStatusResponse, error) {
	return nil, nil
}
func (c *stubCart) Merge(context.Context, uuid.UUID, uuid.UUID, string) (*cart.CartResponse, error) {
	return nil, nil
}
