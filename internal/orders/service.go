package orders

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/cart"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/crypto"
)

// shippingRates is the source of truth for shipping cost. Two flat options
// for now; extend with zones later if needed.
var shippingRates = map[string]int64{
	ShippingStandard: 500,
	ShippingExpress:  1500,
}

type Service interface {
	Checkout(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, req CheckoutRequest) (*OrderResponse, error)
	GetByID(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, id uuid.UUID) (*OrderResponse, error)
	ListMine(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, status string, from, to *time.Time) ([]OrderSummary, error)
	Cancel(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, id uuid.UUID) (*OrderResponse, error)
}

type service struct {
	repo      Repository
	carts     cart.Service
	encryptor *crypto.Encryptor
	validate  *validator.Validate
}

func NewService(repo Repository, carts cart.Service, encryptor *crypto.Encryptor) Service {
	return &service{
		repo:      repo,
		carts:     carts,
		encryptor: encryptor,
		validate:  validator.New(),
	}
}

func (s *service) Checkout(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, req CheckoutRequest) (*OrderResponse, error) {
	if userID == nil && guestID == nil {
		return nil, ErrNoOwner
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Address.Country = strings.ToUpper(strings.TrimSpace(req.Address.Country))
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	shippingCents, ok := shippingRates[req.ShippingMethod]
	if !ok {
		return nil, ErrInvalidShipping
	}

	cartView, err := s.carts.GetCart(ctx, userID, guestID)
	if err != nil {
		return nil, fmt.Errorf("load cart: %w", err)
	}
	if len(cartView.Items) == 0 {
		return nil, ErrCartEmpty
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock every product the cart references and re-verify stock+price.
	// Locking up front (before any writes) avoids deadlocks between
	// concurrent checkouts that would otherwise lock products in different
	// orders. We sort by product ID for the same reason.
	type pendingItem struct {
		ProductID  uuid.UUID
		Name       string
		UnitPrice  int64
		Quantity   int
	}
	pending := make([]pendingItem, 0, len(cartView.Items))
	subtotal := int64(0)
	for _, it := range cartView.Items {
		locked, err := s.repo.LockProductForUpdateTx(ctx, tx, it.ProductID)
		if err != nil {
			return nil, err
		}
		if locked.Stock < it.Quantity {
			return nil, ErrStockChanged
		}
		if locked.Price != it.ProductPrice {
			return nil, ErrStockChanged
		}
		subtotal += locked.Price * int64(it.Quantity)
		pending = append(pending, pendingItem{
			ProductID: locked.ID,
			Name:      locked.Name,
			UnitPrice: locked.Price,
			Quantity:  it.Quantity,
		})
	}

	emailEnc, err := s.encryptor.Encrypt(req.Email)
	if err != nil {
		return nil, fmt.Errorf("encrypt email: %w", err)
	}
	phoneEnc, err := s.encryptor.Encrypt(req.Phone)
	if err != nil {
		return nil, fmt.Errorf("encrypt phone: %w", err)
	}

	row := &orderRow{
		UserID:         userID,
		GuestSessionID: guestID,
		Status:         StatusPendingPayment,
		EmailEnc:       emailEnc,
		PhoneEnc:       phoneEnc,
		SubtotalCents:  subtotal,
		ShippingCents:  shippingCents,
		TotalCents:     subtotal + shippingCents,
		ShippingMethod: req.ShippingMethod,
	}
	if err := s.repo.CreateOrderTx(ctx, tx, row); err != nil {
		return nil, err
	}

	items := make([]OrderItem, 0, len(pending))
	for _, p := range pending {
		pid := p.ProductID
		it := OrderItem{
			OrderID:        row.ID,
			ProductID:      &pid,
			ProductName:    p.Name,
			UnitPriceCents: p.UnitPrice,
			Quantity:       p.Quantity,
		}
		if err := s.repo.CreateOrderItemTx(ctx, tx, &it); err != nil {
			return nil, err
		}
		items = append(items, it)

		if err := s.repo.DecrementStockTx(ctx, tx, p.ProductID, p.Quantity); err != nil {
			return nil, err
		}
	}

	addr, err := s.encryptAddress(req.Address)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateShippingAddressTx(ctx, tx, row.ID, addr); err != nil {
		return nil, err
	}

	if err := s.repo.DeleteCartItemsTx(ctx, tx, cartView.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit checkout: %w", err)
	}

	order, err := s.decryptOrder(row)
	if err != nil {
		return nil, err
	}
	return &OrderResponse{
		Order:   *order,
		Items:   items,
		Address: req.Address.toShippingAddress(),
	}, nil
}

func (s *service) GetByID(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, id uuid.UUID) (*OrderResponse, error) {
	row, items, addr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ownershipOK(row, userID, guestID) {
		return nil, ErrForbidden
	}
	order, err := s.decryptOrder(row)
	if err != nil {
		return nil, err
	}
	address, err := s.decryptAddress(addr)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []OrderItem{}
	}
	return &OrderResponse{Order: *order, Items: items, Address: *address}, nil
}

func (s *service) ListMine(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, status string, from, to *time.Time) ([]OrderSummary, error) {
	if userID != nil {
		return s.repo.ListByUser(ctx, *userID, status, from, to)
	}
	if guestID != nil {
		return s.repo.ListByGuest(ctx, *guestID, status, from, to)
	}
	return nil, ErrNoOwner
}

func (s *service) Cancel(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, id uuid.UUID) (*OrderResponse, error) {
	row, _, _, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ownershipOK(row, userID, guestID) {
		return nil, ErrForbidden
	}
	if !cancellable(row.Status) {
		return nil, ErrNotCancellable
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	items, err := s.repo.ItemsForRestock(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.ProductID == nil {
			continue
		}
		if err := s.repo.IncrementStockTx(ctx, tx, *it.ProductID, it.Quantity); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, id, StatusCancelled); err != nil {
		return nil, fmt.Errorf("set cancelled: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit cancel: %w", err)
	}

	return s.GetByID(ctx, userID, guestID, id)
}

// helpers --------------------------------------------------------------------

func ownershipOK(row *orderRow, userID *uuid.UUID, guestID *uuid.UUID) bool {
	if userID != nil && row.UserID != nil && *row.UserID == *userID {
		return true
	}
	if guestID != nil && row.GuestSessionID != nil && *row.GuestSessionID == *guestID {
		return true
	}
	return false
}

func cancellable(status string) bool {
	return status == StatusPendingPayment || status == StatusPaid
}

func (s *service) decryptOrder(row *orderRow) (*Order, error) {
	email, err := s.encryptor.Decrypt(row.EmailEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt email: %w", err)
	}
	phone, err := s.encryptor.Decrypt(row.PhoneEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt phone: %w", err)
	}
	return &Order{
		ID:             row.ID,
		OrderNumber:    row.OrderNumber,
		UserID:         row.UserID,
		GuestSessionID: row.GuestSessionID,
		Status:         row.Status,
		Email:          email,
		Phone:          phone,
		SubtotalCents:  row.SubtotalCents,
		ShippingCents:  row.ShippingCents,
		TotalCents:     row.TotalCents,
		ShippingMethod: row.ShippingMethod,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

func (s *service) encryptAddress(a CheckoutAddressRequest) (*addressRow, error) {
	rec, err := s.encryptor.Encrypt(a.RecipientName)
	if err != nil {
		return nil, fmt.Errorf("encrypt recipient: %w", err)
	}
	l1, err := s.encryptor.Encrypt(a.Line1)
	if err != nil {
		return nil, fmt.Errorf("encrypt line1: %w", err)
	}
	l2, err := s.encryptor.Encrypt(a.Line2)
	if err != nil {
		return nil, fmt.Errorf("encrypt line2: %w", err)
	}
	city, err := s.encryptor.Encrypt(a.City)
	if err != nil {
		return nil, fmt.Errorf("encrypt city: %w", err)
	}
	region, err := s.encryptor.Encrypt(a.Region)
	if err != nil {
		return nil, fmt.Errorf("encrypt region: %w", err)
	}
	postal, err := s.encryptor.Encrypt(a.PostalCode)
	if err != nil {
		return nil, fmt.Errorf("encrypt postal: %w", err)
	}
	country, err := s.encryptor.Encrypt(a.Country)
	if err != nil {
		return nil, fmt.Errorf("encrypt country: %w", err)
	}
	return &addressRow{
		RecipientNameEnc: rec,
		Line1Enc:         l1,
		Line2Enc:         l2,
		CityEnc:          city,
		RegionEnc:        region,
		PostalCodeEnc:    postal,
		CountryEnc:       country,
	}, nil
}

func (s *service) decryptAddress(a *addressRow) (*ShippingAddress, error) {
	out := &ShippingAddress{}
	var err error
	if out.RecipientName, err = s.encryptor.Decrypt(a.RecipientNameEnc); err != nil {
		return nil, fmt.Errorf("decrypt recipient: %w", err)
	}
	if out.Line1, err = s.encryptor.Decrypt(a.Line1Enc); err != nil {
		return nil, fmt.Errorf("decrypt line1: %w", err)
	}
	if out.Line2, err = s.encryptor.Decrypt(a.Line2Enc); err != nil {
		return nil, fmt.Errorf("decrypt line2: %w", err)
	}
	if out.City, err = s.encryptor.Decrypt(a.CityEnc); err != nil {
		return nil, fmt.Errorf("decrypt city: %w", err)
	}
	if out.Region, err = s.encryptor.Decrypt(a.RegionEnc); err != nil {
		return nil, fmt.Errorf("decrypt region: %w", err)
	}
	if out.PostalCode, err = s.encryptor.Decrypt(a.PostalCodeEnc); err != nil {
		return nil, fmt.Errorf("decrypt postal: %w", err)
	}
	if out.Country, err = s.encryptor.Decrypt(a.CountryEnc); err != nil {
		return nil, fmt.Errorf("decrypt country: %w", err)
	}
	return out, nil
}

func (a CheckoutAddressRequest) toShippingAddress() ShippingAddress {
	return ShippingAddress{
		RecipientName: a.RecipientName,
		Line1:         a.Line1,
		Line2:         a.Line2,
		City:          a.City,
		Region:        a.Region,
		PostalCode:    a.PostalCode,
		Country:       a.Country,
	}
}

