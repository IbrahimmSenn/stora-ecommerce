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

	"github.com/IbrahimmSenn/stora-ecommerce/internal/activity"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/cart"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/crypto"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/metrics"
)

// shippingRates is the built-in fallback used when no ShippingRater is wired
// (e.g. in unit tests). In production the delivery service is the source of
// truth and these two seeded codes mirror its seed rows.
var shippingRates = map[string]int64{
	ShippingStandard: 500,
	ShippingExpress:  1500,
}

// ShippingRater resolves a shipping method code to its cost in cents. ok is
// false for an unknown or inactive method. Implemented by the delivery service
// and injected from main.go (consumer-side interface, like Refunder).
type ShippingRater interface {
	Rate(ctx context.Context, code string) (cents int64, ok bool, err error)
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
	// MarkPaid reports whether the order is in a paid-or-later state after the
	// call. false means the charge has no live order behind it (the order was
	// cancelled/reaped/failed before the payment landed) — the caller must not
	// confirm the purchase.
	MarkPaid(ctx context.Context, id uuid.UUID) (bool, error)
	MarkPaymentFailed(ctx context.Context, id uuid.UUID) error

	// ExpireStaleCheckouts releases stock held by abandoned pending orders.
	// Driven by a background ticker in main.
	ExpireStaleCheckouts(ctx context.Context, olderThan time.Time, limit int) (int, error)

	// Admin entry points (no owner check; guarded by RBAC at the route layer).
	AdminList(ctx context.Context, status string, from, to *time.Time, page, pageSize int) (*AdminOrderList, error)
	AdminGet(ctx context.Context, id uuid.UUID) (*OrderResponse, error)
	AdminUpdateStatus(ctx context.Context, id uuid.UUID, status string) (*OrderResponse, error)
	AdminRefund(ctx context.Context, id uuid.UUID) (*OrderResponse, error)
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

// IntentCanceller voids every still-pending Stripe intent for an order. The
// checkout reaper calls it before releasing reserved stock so an abandoned
// checkout can't be charged after its order is cancelled. Returns an error
// wrapping ErrPaymentInFlight when an intent already succeeded — the order
// must then be reconciled, not reaped. Same consumer-side pattern as Refunder.
type IntentCanceller interface {
	CancelOrderIntents(ctx context.Context, orderID uuid.UUID) error
}

// IntentCancellerFunc adapts a plain function to the IntentCanceller interface.
type IntentCancellerFunc func(ctx context.Context, orderID uuid.UUID) error

func (f IntentCancellerFunc) CancelOrderIntents(ctx context.Context, orderID uuid.UUID) error {
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
	canceller  IntentCanceller
	rater      ShippingRater
	metrics    metrics.Recorder
	activity   activity.Logger
}

type ServiceOption func(*service)

func WithMetrics(r metrics.Recorder) ServiceOption {
	return func(s *service) { s.metrics = r }
}

// WithActivityLogger records a purchase event per order item when an order
// is marked paid — the last stage of the view → add_to_cart → purchase
// funnel in user_activity.
func WithActivityLogger(l activity.Logger) ServiceOption {
	return func(s *service) { s.activity = l }
}

func NewService(repo Repository, carts cart.Service, encryptor *crypto.Encryptor, geocoder Geocoder, refunder Refunder, reconciler Reconciler, canceller IntentCanceller, rater ShippingRater, opts ...ServiceOption) Service {
	v := validator.New()
	// Stricter than `alpha,len=2`: rejects "ZZ" etc. by checking against the
	// real ISO 3166-1 alpha-2 list. Applied via the `iso3166_1_alpha2` tag.
	_ = v.RegisterValidation("iso3166_1_alpha2", func(fl validator.FieldLevel) bool {
		return validCountryCode(fl.Field().String())
	})
	if geocoder == nil {
		geocoder = PassthroughGeocoder{}
	}
	s := &service{
		repo:       repo,
		carts:      carts,
		encryptor:  encryptor,
		validate:   v,
		geocoder:   geocoder,
		refunder:   refunder,
		reconciler: reconciler,
		canceller:  canceller,
		rater:      rater,
		metrics:    metrics.Noop{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// resolveShipping returns the cost for a shipping method code. It uses the
// injected ShippingRater (delivery service) when present, falling back to the
// built-in flat rates otherwise. Unknown/inactive methods yield ErrInvalidShipping.
func (s *service) resolveShipping(ctx context.Context, code string) (int64, error) {
	if s.rater != nil {
		cents, ok, err := s.rater.Rate(ctx, code)
		if err != nil {
			return 0, fmt.Errorf("resolve shipping rate: %w", err)
		}
		if !ok {
			return 0, ErrInvalidShipping
		}
		return cents, nil
	}
	cents, ok := shippingRates[code]
	if !ok {
		return 0, ErrInvalidShipping
	}
	return cents, nil
}

func (s *service) Checkout(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, req CheckoutRequest) (*OrderResponse, error) {
	resp, err := s.checkout(ctx, userID, guestID, req)
	if err != nil {
		s.metrics.CheckoutFailed(checkoutFailReason(err))
		return nil, err
	}
	customerType := "guest"
	if userID != nil {
		customerType = "registered"
	}
	s.metrics.OrderCreated(customerType)
	return resp, nil
}

// checkoutFailReason maps a checkout error to a bounded metric label.
func checkoutFailReason(err error) string {
	var vErr validator.ValidationErrors
	switch {
	case errors.As(err, &vErr), errors.Is(err, ErrNoOwner):
		return "validation"
	case errors.Is(err, ErrAddressNotVerifiable), errors.Is(err, ErrAddressVerificationUnavailable):
		return "address"
	case errors.Is(err, ErrInvalidShipping):
		return "shipping"
	case errors.Is(err, ErrCartEmpty):
		return "empty_cart"
	case errors.Is(err, ErrStockChanged):
		return "stock"
	default:
		return "error"
	}
}

func (s *service) checkout(ctx context.Context, userID *uuid.UUID, guestID *uuid.UUID, req CheckoutRequest) (*OrderResponse, error) {
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

	shippingCents, err := s.resolveShipping(ctx, req.ShippingMethod)
	if err != nil {
		return nil, err
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
		// Compare-and-set first: if a concurrent cancel already moved the order
		// off its cancellable status, this returns false and we restock nothing.
		ok, err := tx.TransitionStatus(ctx, id, row.Status, terminalStatus)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotCancellable
		}
		for _, it := range items {
			if it.ProductID == nil {
				continue
			}
			if err := tx.IncrementStock(ctx, *it.ProductID, it.Quantity); err != nil {
				return err
			}
		}
		return nil
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

// MarkPaid moves a pending order to paid. Idempotent via compare-and-set:
// a replayed webhook finds the order already paid, makes no change, and skips
// re-logging purchases. Only the call that actually flips the status logs.
//
// A no-op CAS is ambiguous — the order may already be paid (webhook replay,
// fine) or it may have left pending_payment another way (reaped, cancelled,
// failed). The returned bool resolves that: true means the order is in a
// paid-or-later state, false means the money has no live order behind it.
func (s *service) MarkPaid(ctx context.Context, id uuid.UUID) (bool, error) {
	ok, err := s.repo.TransitionStatus(ctx, id, StatusPendingPayment, StatusPaid)
	if err != nil {
		return false, err
	}
	if ok {
		s.logPurchases(ctx, id)
		return true, nil
	}
	row, _, _, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	return isPostPaymentStatus(row.Status), nil
}

// isPostPaymentStatus reports whether a status means the order's charge is
// backed by a live, fulfillable order.
func isPostPaymentStatus(status string) bool {
	switch status {
	case StatusPaid, StatusProcessing, StatusShipped, StatusDelivered:
		return true
	}
	return false
}

// logPurchases is best-effort — activity logging never fails an order
// transition (same contract as the activity service itself).
func (s *service) logPurchases(ctx context.Context, id uuid.UUID) {
	if s.activity == nil {
		return
	}
	row, items, _, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Printf("orders: purchase activity for %s skipped: %v", id, err)
		return
	}
	for _, it := range items {
		if it.ProductID == nil {
			continue
		}
		s.activity.LogPurchase(ctx, row.UserID, row.GuestSessionID, it.ProductID, nil)
	}
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
		// Only the transition from pending_payment restocks. If another path
		// (cancel, reaper, a prior failure) already moved the order, this is a
		// no-op so stock is never released twice.
		ok, err := tx.TransitionStatus(ctx, id, StatusPendingPayment, StatusPaymentFailed)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		for _, it := range items {
			if it.ProductID == nil {
				continue
			}
			if err := tx.IncrementStock(ctx, *it.ProductID, it.Quantity); err != nil {
				return err
			}
		}
		return nil
	})
}

// ExpireStaleCheckouts releases the stock reserved by abandoned checkouts:
// orders left in pending_payment since before olderThan are transitioned to
// cancelled and their items restocked. Runs up to limit orders per call.
// Returns the number of orders reaped.
//
// Before touching an order it voids the order's pending Stripe intents, so a
// customer who left the payment form open can no longer be charged for an
// order this is about to cancel. If the canceller reports the payment already
// succeeded (ErrPaymentInFlight), the order isn't abandoned — it's a stuck
// payment (e.g. a lost webhook), so it's handed to the reconciler instead.
// The status transition itself stays a compare-and-set, so even a payment
// that lands mid-reap can't be double-applied.
func (s *service) ExpireStaleCheckouts(ctx context.Context, olderThan time.Time, limit int) (int, error) {
	ids, err := s.repo.StalePendingOrders(ctx, olderThan, limit)
	if err != nil {
		return 0, err
	}
	reaped := 0
	for _, id := range ids {
		if s.canceller != nil {
			if err := s.canceller.CancelOrderIntents(ctx, id); err != nil {
				if errors.Is(err, ErrPaymentInFlight) {
					log.Printf("orders: reaper found a live payment on %s — reconciling instead", id)
					if s.reconciler != nil {
						if rerr := s.reconciler.Reconcile(ctx, id); rerr != nil {
							log.Printf("orders: reconcile of %s failed: %v", id, rerr)
						}
					}
					continue
				}
				log.Printf("orders: reaper skipped %s: cancel intents: %v", id, err)
				continue
			}
		}
		items, err := s.repo.ItemsForRestock(ctx, id)
		if err != nil {
			log.Printf("orders: reaper skipped %s: %v", id, err)
			continue
		}
		flipped := false
		err = s.repo.WithTx(ctx, func(tx TxRepo) error {
			ok, err := tx.TransitionStatus(ctx, id, StatusPendingPayment, StatusCancelled)
			if err != nil {
				return err
			}
			if !ok {
				return nil // already transitioned by payment/cancel — skip restock
			}
			for _, it := range items {
				if it.ProductID == nil {
					continue
				}
				if err := tx.IncrementStock(ctx, *it.ProductID, it.Quantity); err != nil {
					return err
				}
			}
			flipped = true
			return nil
		})
		if err != nil {
			log.Printf("orders: reaper failed on %s: %v", id, err)
			continue
		}
		if flipped {
			reaped++
		}
	}
	return reaped, nil
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
