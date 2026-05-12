package payments

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, p *Payment) error
	GetByPaymentIntentID(ctx context.Context, intentID string) (*Payment, error)
	UpdateSucceeded(ctx context.Context, intentID string) error
	UpdateFailed(ctx context.Context, intentID, code, message string) error
	LatestForOrder(ctx context.Context, orderID uuid.UUID) (*Payment, error)
	MarkRefunded(ctx context.Context, paymentID uuid.UUID, refundID string) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, p *Payment) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO payments (order_id, stripe_payment_intent_id, status, amount_cents, currency)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, created_at, updated_at`,
		p.OrderID, p.StripePaymentIntentID, p.Status, p.AmountCents, p.Currency,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetByPaymentIntentID(ctx context.Context, intentID string) (*Payment, error) {
	var p Payment
	err := r.db.QueryRow(ctx,
		`SELECT id, order_id, stripe_payment_intent_id, status, amount_cents, currency,
			error_code, error_message, stripe_refund_id, refunded_at, created_at, updated_at
		 FROM payments WHERE stripe_payment_intent_id = $1`, intentID,
	).Scan(
		&p.ID, &p.OrderID, &p.StripePaymentIntentID, &p.Status, &p.AmountCents, &p.Currency,
		&p.ErrorCode, &p.ErrorMessage, &p.StripeRefundID, &p.RefundedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("get payment by intent id: %w", err)
	}
	return &p, nil
}

func (r *postgresRepository) UpdateSucceeded(ctx context.Context, intentID string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE payments
		 SET status = $2, error_code = NULL, error_message = NULL
		 WHERE stripe_payment_intent_id = $1`,
		intentID, StatusSucceeded,
	)
	if err != nil {
		return fmt.Errorf("mark payment succeeded: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *postgresRepository) UpdateFailed(ctx context.Context, intentID, code, message string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE payments
		 SET status = $2, error_code = NULLIF($3, ''), error_message = NULLIF($4, '')
		 WHERE stripe_payment_intent_id = $1`,
		intentID, StatusFailed, code, message,
	)
	if err != nil {
		return fmt.Errorf("mark payment failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

// LatestForOrder returns the most recently created payment row for an order,
// or ErrPaymentNotFound when there's nothing yet. Used by the service to
// decide whether a retry or fresh attempt is appropriate.
func (r *postgresRepository) LatestForOrder(ctx context.Context, orderID uuid.UUID) (*Payment, error) {
	var p Payment
	err := r.db.QueryRow(ctx,
		`SELECT id, order_id, stripe_payment_intent_id, status, amount_cents, currency,
			error_code, error_message, stripe_refund_id, refunded_at, created_at, updated_at
		 FROM payments
		 WHERE order_id = $1
		 ORDER BY created_at DESC
		 LIMIT 1`, orderID,
	).Scan(
		&p.ID, &p.OrderID, &p.StripePaymentIntentID, &p.Status, &p.AmountCents, &p.Currency,
		&p.ErrorCode, &p.ErrorMessage, &p.StripeRefundID, &p.RefundedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("latest payment for order: %w", err)
	}
	return &p, nil
}

// MarkRefunded flips the row to refunded, stamping the Stripe refund id and
// time. Idempotent at the SQL level: a row already in 'refunded' is a no-op.
func (r *postgresRepository) MarkRefunded(ctx context.Context, paymentID uuid.UUID, refundID string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE payments
		 SET status = $2, stripe_refund_id = $3, refunded_at = NOW()
		 WHERE id = $1 AND status <> $2`,
		paymentID, StatusRefunded, refundID,
	)
	if err != nil {
		return fmt.Errorf("mark payment refunded: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Either the id doesn't exist or it's already refunded. The service
		// guards against the first case by loading first, so this means
		// idempotent no-op.
		return nil
	}
	return nil
}
