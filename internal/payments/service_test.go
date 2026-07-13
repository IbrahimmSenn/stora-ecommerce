package payments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripego "github.com/stripe/stripe-go/v76"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/orders"
)

func newTestService(repo *stubRepo, ordersSvc *stubOrders, intents *stubIntent, events *stubEvents) Service {
	return newTestServiceWithRefunds(repo, ordersSvc, intents, events, &stubRefunds{})
}

func newTestServiceWithRefunds(repo *stubRepo, ordersSvc *stubOrders, intents *stubIntent, events *stubEvents, refunds *stubRefunds) Service {
	return NewService(repo, ordersSvc, events, intents, refunds, "whsec_test", "pk_test_publishable")
}

func makeOrderResp(id uuid.UUID, status string, total int64, owner *uuid.UUID) *orders.OrderResponse {
	return &orders.OrderResponse{
		Order: orders.Order{
			ID: id, OrderNumber: "ORD-TEST", Status: status,
			Email: "buyer@example.com",
			TotalCents: total, ShippingCents: 500, SubtotalCents: total - 500,
			ShippingMethod: "standard",
			UserID: owner,
		},
		Items: []orders.OrderItem{{
			ID: uuid.New(), ProductName: "Widget", UnitPriceCents: total - 500, Quantity: 1,
		}},
		Address: orders.ShippingAddress{
			RecipientName: "Buyer", Line1: "1 Demo", City: "X", Region: "Y", PostalCode: "0", Country: "US",
		},
	}
}

func TestCreateIntent_RejectsNonPayableStatus(t *testing.T) {
	owner := uuid.New()
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusShipped, 1500, &owner)}
	svc := newTestService(newStubRepo(), ordersSvc, &stubIntent{}, &stubEvents{})

	_, err := svc.CreateIntent(context.Background(), &owner, nil, orderID)
	assert.ErrorIs(t, err, ErrInvalidOrderStatus)
}

func TestCreateIntent_PropagatesOrdersForbidden(t *testing.T) {
	owner := uuid.New()
	orderID := uuid.New()
	ordersSvc := &stubOrders{getErr: orders.ErrForbidden}
	svc := newTestService(newStubRepo(), ordersSvc, &stubIntent{}, &stubEvents{})

	_, err := svc.CreateIntent(context.Background(), &owner, nil, orderID)
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestCreateIntent_PersistsRowAndThreadsMetadata(t *testing.T) {
	owner := uuid.New()
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPendingPayment, 2500, &owner)}
	intents := &stubIntent{nextID: "pi_abc", nextSecret: "pi_abc_secret"}
	repo := newStubRepo()
	svc := newTestService(repo, ordersSvc, intents, &stubEvents{})

	resp, err := svc.CreateIntent(context.Background(), &owner, nil, orderID)
	require.NoError(t, err)
	assert.Equal(t, "pi_abc_secret", resp.ClientSecret)
	assert.Equal(t, "pk_test_publishable", resp.PublishableKey)
	assert.Equal(t, "pi_abc", resp.PaymentIntentID)

	// Stripe call carried the right amount + metadata.
	require.Len(t, intents.calls, 1)
	assert.Equal(t, int64(2500), intents.calls[0].amount)
	assert.Equal(t, "usd", intents.calls[0].currency)
	assert.Equal(t, orderID.String(), intents.calls[0].metadata["order_id"])
	assert.Equal(t, "ORD-TEST", intents.calls[0].metadata["order_number"])

	// Persisted row.
	require.Len(t, repo.byIntent, 1)
	row := repo.byIntent["pi_abc"]
	assert.Equal(t, StatusPending, row.Status)
	assert.Equal(t, int64(2500), row.AmountCents)
}

// A second CreateIntent call for the same order must reuse the existing
// pending PaymentIntent (and fetch its client_secret from Stripe) instead of
// creating a duplicate. Without this guard a double-tap on "Pay" creates
// orphan PIs and orphan payment rows.
func TestCreateIntent_ReusesExistingPendingPayment(t *testing.T) {
	owner := uuid.New()
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPendingPayment, 2500, &owner)}
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_existing",
		Status: StatusPending, AmountCents: 2500, Currency: "usd",
	})
	intents := &stubIntent{
		getStatus: map[string]IntentStatus{
			"pi_existing": {Status: "requires_payment_method", ClientSecret: "pi_existing_secret"},
		},
	}
	svc := newTestService(repo, ordersSvc, intents, &stubEvents{})

	resp, err := svc.CreateIntent(context.Background(), &owner, nil, orderID)
	require.NoError(t, err)
	assert.Equal(t, "pi_existing", resp.PaymentIntentID, "should reuse existing intent")
	assert.Equal(t, "pi_existing_secret", resp.ClientSecret)
	assert.Empty(t, intents.calls, "must not create a new Stripe intent")
	assert.Len(t, repo.byIntent, 1, "must not insert a duplicate payment row")
}

func TestHandleEvent_SucceededFlipsOrderAndPublishes(t *testing.T) {
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPendingPayment, 2500, nil)}
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_xyz",
		Status: StatusPending, AmountCents: 2500, Currency: "usd",
	})
	events := &stubEvents{}
	svc := newTestService(repo, ordersSvc, &stubIntent{}, events).(*service)

	event := stripego.Event{
		Type: "payment_intent.succeeded",
		Data: &stripego.EventData{Raw: json.RawMessage(`{"id":"pi_xyz"}`)},
	}
	require.NoError(t, svc.handleEvent(context.Background(), event))

	assert.Equal(t, StatusSucceeded, repo.byIntent["pi_xyz"].Status)
	assert.Equal(t, []string{orders.StatusPaid}, ordersSvc.statusCalls)
	require.Len(t, events.succeeded, 1)
	assert.Equal(t, orderID, events.succeeded[0].OrderID)
	assert.Equal(t, "pi_xyz", events.succeeded[0].PaymentIntentID)
	assert.Equal(t, int64(2500), events.succeeded[0].AmountCents)
	assert.Empty(t, events.failed)
}

func TestHandleEvent_FailedRecordsErrorAndPublishes(t *testing.T) {
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPendingPayment, 2500, nil)}
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_zzz",
		Status: StatusPending, AmountCents: 2500, Currency: "usd",
	})
	events := &stubEvents{}
	svc := newTestService(repo, ordersSvc, &stubIntent{}, events).(*service)

	event := stripego.Event{
		Type: "payment_intent.payment_failed",
		Data: &stripego.EventData{Raw: json.RawMessage(`{
			"id":"pi_zzz",
			"last_payment_error":{"code":"card_declined","message":"insufficient_funds"}
		}`)},
	}
	require.NoError(t, svc.handleEvent(context.Background(), event))

	row := repo.byIntent["pi_zzz"]
	assert.Equal(t, StatusFailed, row.Status)
	require.NotNil(t, row.ErrorCode)
	assert.Equal(t, "card_declined", *row.ErrorCode)
	require.NotNil(t, row.ErrorMessage)
	assert.Equal(t, "insufficient_funds", *row.ErrorMessage)
	assert.Equal(t, []string{orders.StatusPaymentFailed}, ordersSvc.statusCalls)
	require.Len(t, events.failed, 1)
	assert.Equal(t, "card_declined", events.failed[0].FailureCode)
	assert.Equal(t, "insufficient_funds", events.failed[0].FailureMessage)
	assert.Empty(t, events.succeeded)
}

func TestHandleEvent_IdempotentOnDuplicate(t *testing.T) {
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPaid, 2500, nil)}
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_dup",
		Status: StatusSucceeded, AmountCents: 2500, Currency: "usd",
	})
	events := &stubEvents{}
	svc := newTestService(repo, ordersSvc, &stubIntent{}, events).(*service)

	event := stripego.Event{
		Type: "payment_intent.succeeded",
		Data: &stripego.EventData{Raw: json.RawMessage(`{"id":"pi_dup"}`)},
	}
	require.NoError(t, svc.handleEvent(context.Background(), event))

	// No status flip, no extra publish — Stripe retried, we no-op'd.
	assert.Empty(t, ordersSvc.statusCalls)
	assert.Empty(t, events.succeeded)
	assert.Empty(t, events.failed)
}

func TestHandleWebhook_BadSignature(t *testing.T) {
	svc := newTestService(newStubRepo(), &stubOrders{}, &stubIntent{}, &stubEvents{})

	err := svc.HandleWebhook(context.Background(), []byte(`{}`), "garbage")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureMismatch)
}

func TestRefundOrder_HappyPath(t *testing.T) {
	orderID := uuid.New()
	paymentID := uuid.New()
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: paymentID, OrderID: orderID, StripePaymentIntentID: "pi_succ",
		Status: StatusSucceeded, AmountCents: 2500, Currency: "usd",
	})
	refunds := &stubRefunds{nextID: "re_abc"}
	svc := newTestServiceWithRefunds(repo, &stubOrders{}, &stubIntent{}, &stubEvents{}, refunds)

	require.NoError(t, svc.RefundOrder(context.Background(), orderID))

	require.Len(t, refunds.calls, 1)
	assert.Equal(t, "pi_succ", refunds.calls[0].intentID)
	assert.Equal(t, paymentID.String(), refunds.calls[0].idempotencyKey)

	row := repo.byIntent["pi_succ"]
	assert.Equal(t, StatusRefunded, row.Status)
	require.NotNil(t, row.StripeRefundID)
	assert.Equal(t, "re_abc", *row.StripeRefundID)
}

func TestRefundOrder_IdempotentWhenAlreadyRefunded(t *testing.T) {
	orderID := uuid.New()
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_done",
		Status: StatusRefunded, AmountCents: 2500, Currency: "usd",
	})
	refunds := &stubRefunds{}
	svc := newTestServiceWithRefunds(repo, &stubOrders{}, &stubIntent{}, &stubEvents{}, refunds)

	require.NoError(t, svc.RefundOrder(context.Background(), orderID))
	assert.Empty(t, refunds.calls, "should not call Stripe when already refunded")
}

func TestRefundOrder_RejectsUnpaidPayment(t *testing.T) {
	orderID := uuid.New()
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_pend",
		Status: StatusPending, AmountCents: 2500, Currency: "usd",
	})
	refunds := &stubRefunds{}
	svc := newTestServiceWithRefunds(repo, &stubOrders{}, &stubIntent{}, &stubEvents{}, refunds)

	err := svc.RefundOrder(context.Background(), orderID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCannotRefund)
	assert.Empty(t, refunds.calls)
}

func TestRefundOrder_NoPaymentForOrder(t *testing.T) {
	repo := newStubRepo()
	svc := newTestService(repo, &stubOrders{}, &stubIntent{}, &stubEvents{})

	err := svc.RefundOrder(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrPaymentNotFound)
}

func TestRefundOrder_PropagatesStripeError(t *testing.T) {
	orderID := uuid.New()
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_x",
		Status: StatusSucceeded, AmountCents: 2500, Currency: "usd",
	})
	refunds := &stubRefunds{err: errors.New("stripe down")}
	svc := newTestServiceWithRefunds(repo, &stubOrders{}, &stubIntent{}, &stubEvents{}, refunds)

	err := svc.RefundOrder(context.Background(), orderID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stripe down")
	// DB still shows succeeded — caller retries safely.
	assert.Equal(t, StatusSucceeded, repo.byIntent["pi_x"].Status)
}

func TestReconcile_StripeSaysSucceeded_FlipsOrder(t *testing.T) {
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPendingPayment, 7300, nil)}
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_recon",
		Status: StatusPending, AmountCents: 7300, Currency: "usd",
	})
	intents := &stubIntent{getStatus: map[string]IntentStatus{"pi_recon": {Status: "succeeded"}}}
	events := &stubEvents{}
	svc := newTestService(repo, ordersSvc, intents, events)

	require.NoError(t, svc.Reconcile(context.Background(), orderID))

	assert.Equal(t, []string{"pi_recon"}, intents.getCalls)
	assert.Equal(t, StatusSucceeded, repo.byIntent["pi_recon"].Status)
	assert.Equal(t, []string{orders.StatusPaid}, ordersSvc.statusCalls)
	require.Len(t, events.succeeded, 1)
	assert.Equal(t, orderID, events.succeeded[0].OrderID)
	assert.Empty(t, events.failed)
}

func TestReconcile_StripeSaysCanceled_FlipsOrderToFailed(t *testing.T) {
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPendingPayment, 4200, nil)}
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_canc",
		Status: StatusPending, AmountCents: 4200, Currency: "usd",
	})
	intents := &stubIntent{getStatus: map[string]IntentStatus{"pi_canc": {
		Status:        "canceled",
		LastErrorCode: "abandoned",
		LastErrorMsg:  "intent canceled",
	}}}
	events := &stubEvents{}
	svc := newTestService(repo, ordersSvc, intents, events)

	require.NoError(t, svc.Reconcile(context.Background(), orderID))

	assert.Equal(t, StatusFailed, repo.byIntent["pi_canc"].Status)
	assert.Equal(t, []string{orders.StatusPaymentFailed}, ordersSvc.statusCalls)
	require.Len(t, events.failed, 1)
	assert.Equal(t, "abandoned", events.failed[0].FailureCode)
}

func TestReconcile_StripeStillProcessing_NoSideEffects(t *testing.T) {
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPendingPayment, 4200, nil)}
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_proc",
		Status: StatusPending, AmountCents: 4200, Currency: "usd",
	})
	intents := &stubIntent{getStatus: map[string]IntentStatus{"pi_proc": {Status: "processing"}}}
	events := &stubEvents{}
	svc := newTestService(repo, ordersSvc, intents, events)

	require.NoError(t, svc.Reconcile(context.Background(), orderID))

	assert.Equal(t, StatusPending, repo.byIntent["pi_proc"].Status)
	assert.Empty(t, ordersSvc.statusCalls)
	assert.Empty(t, events.succeeded)
	assert.Empty(t, events.failed)
}

func TestReconcile_AlreadyTerminal_SkipsStripe(t *testing.T) {
	orderID := uuid.New()
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_term",
		Status: StatusSucceeded, AmountCents: 4200, Currency: "usd",
	})
	intents := &stubIntent{}
	svc := newTestService(repo, &stubOrders{}, intents, &stubEvents{})

	require.NoError(t, svc.Reconcile(context.Background(), orderID))
	assert.Empty(t, intents.getCalls, "should not call Stripe when payment is already terminal")
}

func TestReconcile_NoPaymentRow_NoOp(t *testing.T) {
	intents := &stubIntent{}
	svc := newTestService(newStubRepo(), &stubOrders{}, intents, &stubEvents{})

	require.NoError(t, svc.Reconcile(context.Background(), uuid.New()))
	assert.Empty(t, intents.getCalls)
}

func TestHandleEvent_UnhandledTypeIsNoop(t *testing.T) {
	svc := newTestService(newStubRepo(), &stubOrders{}, &stubIntent{}, &stubEvents{}).(*service)

	event := stripego.Event{
		Type: "charge.refunded",
		Data: &stripego.EventData{Raw: json.RawMessage(`{}`)},
	}
	assert.NoError(t, svc.handleEvent(context.Background(), event))
}

// --- stubs ---

type stubRepo struct {
	byIntent map[string]*Payment
}

func newStubRepo() *stubRepo {
	return &stubRepo{byIntent: map[string]*Payment{}}
}

func (s *stubRepo) seed(p *Payment) { s.byIntent[p.StripePaymentIntentID] = p }

func (s *stubRepo) Create(_ context.Context, p *Payment) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
		p.UpdatedAt = p.CreatedAt
	}
	c := *p
	s.byIntent[p.StripePaymentIntentID] = &c
	return nil
}

func (s *stubRepo) GetByPaymentIntentID(_ context.Context, intentID string) (*Payment, error) {
	if p, ok := s.byIntent[intentID]; ok {
		return p, nil
	}
	return nil, ErrPaymentNotFound
}

func (s *stubRepo) UpdateSucceeded(_ context.Context, intentID string) error {
	p, ok := s.byIntent[intentID]
	if !ok {
		return ErrPaymentNotFound
	}
	p.Status = StatusSucceeded
	p.ErrorCode, p.ErrorMessage = nil, nil
	return nil
}

func (s *stubRepo) UpdateFailed(_ context.Context, intentID, code, message string) error {
	p, ok := s.byIntent[intentID]
	if !ok {
		return ErrPaymentNotFound
	}
	p.Status = StatusFailed
	if code != "" {
		c := code
		p.ErrorCode = &c
	}
	if message != "" {
		m := message
		p.ErrorMessage = &m
	}
	return nil
}

func (s *stubRepo) LatestForOrder(_ context.Context, orderID uuid.UUID) (*Payment, error) {
	for _, p := range s.byIntent {
		if p.OrderID == orderID {
			return p, nil
		}
	}
	return nil, ErrPaymentNotFound
}

func (s *stubRepo) MarkRefunded(_ context.Context, paymentID uuid.UUID, refundID string) error {
	for _, p := range s.byIntent {
		if p.ID == paymentID {
			p.Status = StatusRefunded
			rid := refundID
			p.StripeRefundID = &rid
			return nil
		}
	}
	return ErrPaymentNotFound
}

type stubRefundCall struct {
	intentID       string
	idempotencyKey string
}

type stubRefunds struct {
	calls    []stubRefundCall
	nextID   string
	err      error
}

func (s *stubRefunds) Refund(_ context.Context, intentID, idempotencyKey string) (string, error) {
	s.calls = append(s.calls, stubRefundCall{intentID, idempotencyKey})
	if s.err != nil {
		return "", s.err
	}
	id := s.nextID
	if id == "" {
		id = "re_stub"
	}
	return id, nil
}

type stubOrders struct {
	order       *orders.OrderResponse
	getErr      error
	statusCalls []string
}

func (s *stubOrders) Checkout(context.Context, *uuid.UUID, *uuid.UUID, orders.CheckoutRequest) (*orders.OrderResponse, error) {
	return nil, errors.New("not used")
}
func (s *stubOrders) GetByID(_ context.Context, _, _ *uuid.UUID, _ uuid.UUID) (*orders.OrderResponse, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.order, nil
}
func (s *stubOrders) ListMine(context.Context, *uuid.UUID, *uuid.UUID, string, *time.Time, *time.Time) ([]orders.OrderSummary, error) {
	return nil, nil
}
func (s *stubOrders) Cancel(context.Context, *uuid.UUID, *uuid.UUID, uuid.UUID) (*orders.OrderResponse, error) {
	return nil, nil
}
func (s *stubOrders) GetLatestPrefill(context.Context, uuid.UUID) (*orders.PrefillResponse, error) {
	return nil, nil
}
func (s *stubOrders) LoadByID(_ context.Context, _ uuid.UUID) (*orders.OrderResponse, error) {
	return s.order, nil
}
func (s *stubOrders) MarkPaid(_ context.Context, _ uuid.UUID) error {
	s.statusCalls = append(s.statusCalls, orders.StatusPaid)
	return nil
}
func (s *stubOrders) MarkPaymentFailed(_ context.Context, _ uuid.UUID) error {
	s.statusCalls = append(s.statusCalls, orders.StatusPaymentFailed)
	return nil
}
func (s *stubOrders) ExpireStaleCheckouts(context.Context, time.Time, int) (int, error) {
	return 0, nil
}
func (s *stubOrders) AdminList(context.Context, string, *time.Time, *time.Time, int, int) (*orders.AdminOrderList, error) {
	return &orders.AdminOrderList{}, nil
}
func (s *stubOrders) AdminGet(_ context.Context, _ uuid.UUID) (*orders.OrderResponse, error) {
	return nil, orders.ErrOrderNotFound
}
func (s *stubOrders) AdminUpdateStatus(_ context.Context, _ uuid.UUID, _ string) (*orders.OrderResponse, error) {
	return nil, orders.ErrOrderNotFound
}
func (s *stubOrders) AdminRefund(_ context.Context, _ uuid.UUID) (*orders.OrderResponse, error) {
	return nil, orders.ErrOrderNotFound
}

type stubIntentCall struct {
	amount   int64
	currency string
	metadata map[string]string
}

type stubIntent struct {
	nextID     string
	nextSecret string
	calls      []stubIntentCall
	// getStatus is the IntentStatus returned by GetIntent. Tests that exercise
	// Reconcile populate this per-intent-id; unknown ids default to "processing"
	// so reconcile is a no-op unless explicitly programmed.
	getStatus map[string]IntentStatus
	getErr    error
	getCalls  []string
}

func (s *stubIntent) NewIntent(_ context.Context, amount int64, currency string, metadata map[string]string) (string, string, error) {
	s.calls = append(s.calls, stubIntentCall{amount, currency, metadata})
	id := s.nextID
	if id == "" {
		id = "pi_stub"
	}
	secret := s.nextSecret
	if secret == "" {
		secret = id + "_secret"
	}
	return id, secret, nil
}

func (s *stubIntent) GetIntent(_ context.Context, id string) (IntentStatus, error) {
	s.getCalls = append(s.getCalls, id)
	if s.getErr != nil {
		return IntentStatus{}, s.getErr
	}
	if v, ok := s.getStatus[id]; ok {
		return v, nil
	}
	return IntentStatus{Status: "processing"}, nil
}

type stubEvents struct {
	succeeded []PaymentSucceededEvent
	failed    []PaymentFailedEvent
}

func (s *stubEvents) PublishSucceeded(_ context.Context, evt PaymentSucceededEvent) error {
	s.succeeded = append(s.succeeded, evt)
	return nil
}

func (s *stubEvents) PublishFailed(_ context.Context, evt PaymentFailedEvent) error {
	s.failed = append(s.failed, evt)
	return nil
}
