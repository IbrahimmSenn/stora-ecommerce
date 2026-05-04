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

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/orders"
)

func newTestService(repo *stubRepo, ordersSvc *stubOrders, intents *stubIntent, mail *stubMailer) Service {
	return NewService(repo, ordersSvc, mail, intents, "whsec_test", "pk_test_publishable")
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
	svc := newTestService(newStubRepo(), ordersSvc, &stubIntent{}, &stubMailer{})

	_, err := svc.CreateIntent(context.Background(), &owner, nil, orderID)
	assert.ErrorIs(t, err, ErrInvalidOrderStatus)
}

func TestCreateIntent_PropagatesOrdersForbidden(t *testing.T) {
	owner := uuid.New()
	orderID := uuid.New()
	ordersSvc := &stubOrders{getErr: orders.ErrForbidden}
	svc := newTestService(newStubRepo(), ordersSvc, &stubIntent{}, &stubMailer{})

	_, err := svc.CreateIntent(context.Background(), &owner, nil, orderID)
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestCreateIntent_PersistsRowAndThreadsMetadata(t *testing.T) {
	owner := uuid.New()
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPendingPayment, 2500, &owner)}
	intents := &stubIntent{nextID: "pi_abc", nextSecret: "pi_abc_secret"}
	repo := newStubRepo()
	svc := newTestService(repo, ordersSvc, intents, &stubMailer{})

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

func TestHandleEvent_SucceededFlipsOrderAndSendsEmail(t *testing.T) {
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPendingPayment, 2500, nil)}
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_xyz",
		Status: StatusPending, AmountCents: 2500, Currency: "usd",
	})
	mail := &stubMailer{}
	svc := newTestService(repo, ordersSvc, &stubIntent{}, mail).(*service)

	event := stripego.Event{
		Type: "payment_intent.succeeded",
		Data: &stripego.EventData{Raw: json.RawMessage(`{"id":"pi_xyz"}`)},
	}
	require.NoError(t, svc.handleEvent(context.Background(), event))

	assert.Equal(t, StatusSucceeded, repo.byIntent["pi_xyz"].Status)
	assert.Equal(t, []string{orders.StatusPaid}, ordersSvc.statusCalls)
	require.Len(t, mail.sent, 1)
	assert.Equal(t, "buyer@example.com", mail.sent[0].to)
	assert.Contains(t, mail.sent[0].subject, "ORD-TEST")
	assert.Contains(t, mail.sent[0].subject, "received")
}

func TestHandleEvent_FailedRecordsErrorAndEmails(t *testing.T) {
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPendingPayment, 2500, nil)}
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_zzz",
		Status: StatusPending, AmountCents: 2500, Currency: "usd",
	})
	mail := &stubMailer{}
	svc := newTestService(repo, ordersSvc, &stubIntent{}, mail).(*service)

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
	require.Len(t, mail.sent, 1)
	assert.Contains(t, mail.sent[0].subject, "failed")
}

func TestHandleEvent_IdempotentOnDuplicate(t *testing.T) {
	orderID := uuid.New()
	ordersSvc := &stubOrders{order: makeOrderResp(orderID, orders.StatusPaid, 2500, nil)}
	repo := newStubRepo()
	repo.seed(&Payment{
		ID: uuid.New(), OrderID: orderID, StripePaymentIntentID: "pi_dup",
		Status: StatusSucceeded, AmountCents: 2500, Currency: "usd",
	})
	mail := &stubMailer{}
	svc := newTestService(repo, ordersSvc, &stubIntent{}, mail).(*service)

	event := stripego.Event{
		Type: "payment_intent.succeeded",
		Data: &stripego.EventData{Raw: json.RawMessage(`{"id":"pi_dup"}`)},
	}
	require.NoError(t, svc.handleEvent(context.Background(), event))

	// No status flip, no extra email — Stripe retried, we no-op'd.
	assert.Empty(t, ordersSvc.statusCalls)
	assert.Empty(t, mail.sent)
}

func TestHandleWebhook_BadSignature(t *testing.T) {
	svc := newTestService(newStubRepo(), &stubOrders{}, &stubIntent{}, &stubMailer{})

	err := svc.HandleWebhook(context.Background(), []byte(`{}`), "garbage")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSignatureMismatch)
}

func TestHandleEvent_UnhandledTypeIsNoop(t *testing.T) {
	svc := newTestService(newStubRepo(), &stubOrders{}, &stubIntent{}, &stubMailer{}).(*service)

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

type stubIntentCall struct {
	amount   int64
	currency string
	metadata map[string]string
}

type stubIntent struct {
	nextID     string
	nextSecret string
	calls      []stubIntentCall
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

type sentEmail struct {
	to      string
	subject string
	body    string
}

type stubMailer struct {
	sent []sentEmail
}

func (m *stubMailer) Send(to, subject, body string) error {
	m.sent = append(m.sent, sentEmail{to, subject, body})
	return nil
}
