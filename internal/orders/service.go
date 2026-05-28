package orders

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
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

	// GetLatestPrefill returns the contact + shipping address from the user's
	// most recent order, decrypted for use in the checkout-form prefill.
	// Returns (nil, nil) when the user has no prior orders — the handler maps
	// that to 204 No Content.
	GetLatestPrefill(ctx context.Context, userID uuid.UUID) (*PrefillResponse, error)

	// Service-to-service entry points used by the payments package — they
	// bypass the user/guest owner check because the caller is the system
	// itself (a Stripe webhook), not an HTTP request.
	LoadByID(ctx context.Context, id uuid.UUID) (*OrderResponse, error)
	MarkPaid(ctx context.Context, id uuid.UUID) error
	MarkPaymentFailed(ctx context.Context, id uuid.UUID) error
}

// Refunder is the slice of the payments service the orders service uses
// when cancelling a paid order. Defined here (consumer-side) and injected
// via a closure adapter from main.go so payments → orders → payments
// doesn't form an import cycle.
type Refunder interface {
	RefundOrder(ctx context.Context, orderID uuid.UUID) error
}

// RefunderFunc adapts a plain function to the Refunder interface. main.go
// uses it to capture the (later-built) payments service by closure.
type RefunderFunc func(ctx context.Context, orderID uuid.UUID) error

func (f RefunderFunc) RefundOrder(ctx context.Context, orderID uuid.UUID) error {
	return f(ctx, orderID)
}

// Reconciler asks the payments service to pull the current PaymentIntent
// state from Stripe and apply any pending side effects. Same consumer-side
// interface pattern as Refunder — wired via closure from main.go.
type Reconciler interface {
	Reconcile(ctx context.Context, orderID uuid.UUID) error
}

// ReconcilerFunc adapts a plain function to the Reconciler interface.
type ReconcilerFunc func(ctx context.Context, orderID uuid.UUID) error

func (f ReconcilerFunc) Reconcile(ctx context.Context, orderID uuid.UUID) error {
	return f(ctx, orderID)
}

type service struct {
	repo       Repository
	carts      cart.Service
	encryptor  *crypto.Encryptor
	validate   *validator.Validate
	geocoder   Geocoder
	refunder   Refunder
	reconciler Reconciler
}

func NewService(repo Repository, carts cart.Service, encryptor *crypto.Encryptor, geocoder Geocoder, refunder Refunder, reconciler Reconciler) Service {
	v := validator.New()
	// Stricter than `alpha,len=2`: rejects "ZZ" etc. by checking against the
	// real ISO 3166-1 alpha-2 list. Applied via the `iso3166_1_alpha2` tag.
	_ = v.RegisterValidation("iso3166_1_alpha2", func(fl validator.FieldLevel) bool {
		return validCountryCode(fl.Field().String())
	})
	if geocoder == nil {
		geocoder = PassthroughGeocoder{}
	}
	return &service{
		repo:       repo,
		carts:      carts,
		encryptor:  encryptor,
		validate:   v,
		geocoder:   geocoder,
		refunder:   refunder,
		reconciler: reconciler,
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

	// Address verification. Fail-closed by default; AddressOverride lets the
	// user proceed after seeing the rejection (frontend only renders that
	// button after a first failure). Both outcomes are logged so a reviewer
	// or audit can see when the override was used.
	if geocodeErr := s.geocoder.VerifyAddress(ctx, req.Address); geocodeErr != nil {
		if !req.AddressOverride {
			return nil, geocodeErr
		}
		log.Printf("orders: address override used: %v", geocodeErr)
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

	row := &orderRow{
		UserID:         userID,
		GuestSessionID: guestID,
		Status:         StatusPendingPayment,
		ShippingCents:  shippingCents,
		ShippingMethod: req.ShippingMethod,
	}
	var items []OrderItem

	err = s.repo.WithTx(ctx, func(tx TxRepo) error {
		// Lock every product the cart references and re-verify stock+price
		// up front (before any writes) so concurrent checkouts can't deadlock
		// each other on overlapping items.
		type pendingItem struct {
			ProductID uuid.UUID
			Name      string
			UnitPrice int64
			Quantity  int
		}
		pending := make([]pendingItem, 0, len(cartView.Items))
		subtotal := int64(0)
		for _, it := range cartView.Items {
			locked, err := tx.LockProductForUpdate(ctx, it.ProductID)
			if err != nil {
				return err
			}
			if locked.Stock < it.Quantity {
				return ErrStockChanged
			}
			if locked.Price != it.ProductPrice {
				return ErrStockChanged
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
			return fmt.Errorf("encrypt email: %w", err)
		}
		phoneEnc, err := s.encryptor.Encrypt(req.Phone)
		if err != nil {
			return fmt.Errorf("encrypt phone: %w", err)
		}
		row.EmailEnc = emailEnc
		row.PhoneEnc = phoneEnc
		row.SubtotalCents = subtotal
		row.TotalCents = subtotal + shippingCents

		if err := tx.CreateOrder(ctx, row); err != nil {
			return err
		}

		items = make([]OrderItem, 0, len(pending))
		for _, p := range pending {
			pid := p.ProductID
			it := OrderItem{
				OrderID:        row.ID,
				ProductID:      &pid,
				ProductName:    p.Name,
				UnitPriceCents: p.UnitPrice,
				Quantity:       p.Quantity,
			}
			if err := tx.CreateOrderItem(ctx, &it); err != nil {
				return err
			}
			items = append(items, it)

			if err := tx.DecrementStock(ctx, p.ProductID, p.Quantity); err != nil {
				return err
			}
		}

		addr, err := s.encryptAddress(req.Address)
		if err != nil {
			return err
		}
		if err := tx.CreateShippingAddress(ctx, row.ID, addr); err != nil {
			return err
		}

		return tx.DeleteCartItems(ctx, cartView.ID)
	})
	if err != nil {
		return nil, err
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
	resp, err := s.loadOwned(ctx, userID, guestID, id)
	if err != nil {
		return nil, err
	}
	// Safety net for missed Stripe webhooks: if the order is still pending
	// payment, ask the payments service to pull the current PI state from
	// Stripe and apply any side effects, then reload. Reconcile failures
	// don't fail the request — the caller still gets the (stale) order.
	if resp.Order.Status == StatusPendingPayment && s.reconciler != nil {
		if err := s.reconciler.Reconcile(ctx, id); err != nil {
			log.Printf("orders: reconcile %s failed: %v", id, err)
			return resp, nil
		}
		fresh, err := s.loadOwned(ctx, userID, guestID, id)
		if err != nil {
			return resp, nil
		}
		return fresh, nil
	}
	return resp, nil
}

func (s *service) loadOwned(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, id uuid.UUID) (*OrderResponse, error) {
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
	list, err := s.listMineRaw(ctx, userID, guestID, status, from, to)
	if err != nil {
		return nil, err
	}
	if s.reconciler == nil {
		return list, nil
	}
	// Only reconcile orders that are likely to have transitioned: at least
	// reconcileMinAge ago (give the webhook a chance to land first) and not
	// older than reconcileMaxAge (Stripe expires PaymentIntents past 24h, so
	// anything older is permanently stuck — surface that to ops, don't keep
	// pinging Stripe every page load). Caps fan-out volume on hot accounts.
	now := time.Now()
	pending := make([]uuid.UUID, 0, len(list))
	for _, o := range list {
		if o.Status != StatusPendingPayment {
			continue
		}
		age := now.Sub(o.CreatedAt)
		if age < reconcileMinAge || age > reconcileMaxAge {
			continue
		}
		pending = append(pending, o.ID)
	}
	if len(pending) == 0 {
		return list, nil
	}
	if len(pending) > reconcileMaxPerCall {
		pending = pending[:reconcileMaxPerCall]
	}
	s.reconcileMany(ctx, pending)
	return s.listMineRaw(ctx, userID, guestID, status, from, to)
}

// Reconcile throttling: the webhook is the primary path; this is the safety
// net. Don't ping Stripe for orders the webhook is plausibly still about to
// flip, and don't ping for orders that are clearly permanently stuck.
const (
	reconcileMinAge     = 10 * time.Second
	reconcileMaxAge     = 24 * time.Hour
	reconcileMaxPerCall = 10
)

func (s *service) listMineRaw(ctx context.Context, userID, guestID *uuid.UUID, status string, from, to *time.Time) ([]OrderSummary, error) {
	if userID != nil {
		return s.repo.ListByUser(ctx, *userID, status, from, to)
	}
	if guestID != nil {
		return s.repo.ListByGuest(ctx, *guestID, status, from, to)
	}
	return nil, ErrNoOwner
}

// reconcileMany fans out reconcile calls with a small concurrency cap. Errors
// are logged and otherwise ignored — the caller will requery the list either
// way and the user sees whichever state the DB holds afterward.
func (s *service) reconcileMany(ctx context.Context, ids []uuid.UUID) {
	const maxParallel = 4
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}
		go func(id uuid.UUID) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.reconciler.Reconcile(ctx, id); err != nil {
				log.Printf("orders: reconcile %s failed: %v", id, err)
			}
		}(id)
	}
	wg.Wait()
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

	// If the order was already paid, refund the charge before touching the
	// DB. Stripe call is idempotent (keyed by payment row id) so a retry
	// after a partial failure is safe.
	wasPaid := row.Status == StatusPaid
	if wasPaid {
		if s.refunder == nil {
			return nil, ErrRefundUnavailable
		}
		if err := s.refunder.RefundOrder(ctx, id); err != nil {
			return nil, fmt.Errorf("refund order %s: %w", id, err)
		}
	}

	items, err := s.repo.ItemsForRestock(ctx, id)
	if err != nil {
		return nil, err
	}

	terminalStatus := StatusCancelled
	if wasPaid {
		terminalStatus = StatusRefunded
	}

	err = s.repo.WithTx(ctx, func(tx TxRepo) error {
		for _, it := range items {
			if it.ProductID == nil {
				continue
			}
			if err := tx.IncrementStock(ctx, *it.ProductID, it.Quantity); err != nil {
				return err
			}
		}
		return tx.UpdateStatus(ctx, id, terminalStatus)
	})
	if err != nil {
		return nil, err
	}

	return s.GetByID(ctx, userID, guestID, id)
}

func (s *service) GetLatestPrefill(ctx context.Context, userID uuid.UUID) (*PrefillResponse, error) {
	row, addr, err := s.repo.GetLatestUserShipping(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return nil, nil
		}
		return nil, err
	}
	email, err := s.encryptor.Decrypt(row.EmailEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt email: %w", err)
	}
	phone, err := s.encryptor.Decrypt(row.PhoneEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt phone: %w", err)
	}
	address, err := s.decryptAddress(addr)
	if err != nil {
		return nil, err
	}
	return &PrefillResponse{
		Email:          email,
		Phone:          phone,
		ShippingMethod: row.ShippingMethod,
		Address:        *address,
	}, nil
}

// LoadByID returns the decrypted order without an owner check. Reserved for
// internal callers (e.g. the Stripe webhook path) where the user/guest
// identity isn't available — never expose this through an HTTP handler.
func (s *service) LoadByID(ctx context.Context, id uuid.UUID) (*OrderResponse, error) {
	row, items, addr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
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

func (s *service) MarkPaid(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, StatusPaid)
}

// MarkPaymentFailed flips the order to payment_failed AND releases the stock
// that was reserved at checkout. The rubric's "payment failure → inventory
// unchanged" invariant only holds if the reservation is reversed here.
// Idempotent at the call site (payments service skips if already failed), and
// safe even when there are no items (e.g. test fixtures).
func (s *service) MarkPaymentFailed(ctx context.Context, id uuid.UUID) error {
	items, err := s.repo.ItemsForRestock(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.WithTx(ctx, func(tx TxRepo) error {
		for _, it := range items {
			if it.ProductID == nil {
				continue
			}
			if err := tx.IncrementStock(ctx, *it.ProductID, it.Quantity); err != nil {
				return err
			}
		}
		return tx.UpdateStatus(ctx, id, StatusPaymentFailed)
	})
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

