package payments

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/crypto"
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
	db        *pgxpool.Pool
	encryptor *crypto.Encryptor
}

// NewRepository builds a payments repository that encrypts sensitive columns
// at rest (Stripe identifiers, error messages) and uses an HMAC fingerprint
// for equality lookup on the payment intent id.
func NewRepository(db *pgxpool.Pool, encryptor *crypto.Encryptor) Repository {
	return &postgresRepository{db: db, encryptor: encryptor}
}

func (r *postgresRepository) Create(ctx context.Context, p *Payment) error {
	piEnc, err := r.encryptor.Encrypt(p.StripePaymentIntentID)
	if err != nil {
		return fmt.Errorf("encrypt payment intent id: %w", err)
	}
	piHMAC := r.encryptor.HMAC(p.StripePaymentIntentID)

	err = r.db.QueryRow(ctx,
		`INSERT INTO payments (
			order_id, stripe_payment_intent_id_enc, stripe_payment_intent_id_hmac,
			status, amount_cents, currency
		 ) VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id, created_at, updated_at`,
		p.OrderID, piEnc, piHMAC, p.Status, p.AmountCents, p.Currency,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

// scanRow reads an encrypted payment row from the cursor and decrypts the
// sensitive columns into the returned Payment. Shared by every SELECT.
func (r *postgresRepository) scanRow(row pgx.Row) (*Payment, error) {
	var p Payment
	var piEnc, refundEnc, codeEnc, msgEnc []byte
	err := row.Scan(
		&p.ID, &p.OrderID, &piEnc, &p.Status, &p.AmountCents, &p.Currency,
		&codeEnc, &msgEnc, &refundEnc, &p.RefundedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if p.StripePaymentIntentID, err = r.encryptor.Decrypt(piEnc); err != nil {
		return nil, fmt.Errorf("decrypt payment intent id: %w", err)
	}
	refund, err := r.encryptor.Decrypt(refundEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt refund id: %w", err)
	}
	if refund != "" {
		p.StripeRefundID = &refund
	}
	code, err := r.encryptor.Decrypt(codeEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt error code: %w", err)
	}
	if code != "" {
		p.ErrorCode = &code
	}
	msg, err := r.encryptor.Decrypt(msgEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt error message: %w", err)
	}
	if msg != "" {
		p.ErrorMessage = &msg
	}
	return &p, nil
}

const selectColumns = `id, order_id, stripe_payment_intent_id_enc, status, amount_cents, currency,
	error_code_enc, error_message_enc, stripe_refund_id_enc, refunded_at, created_at, updated_at`

func (r *postgresRepository) GetByPaymentIntentID(ctx context.Context, intentID string) (*Payment, error) {
	piHMAC := r.encryptor.HMAC(intentID)
	row := r.db.QueryRow(ctx,
		`SELECT `+selectColumns+`
		 FROM payments WHERE stripe_payment_intent_id_hmac = $1`,
		piHMAC,
	)
	p, err := r.scanRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("get payment by intent id: %w", err)
	}
	return p, nil
}

func (r *postgresRepository) UpdateSucceeded(ctx context.Context, intentID string) error {
	piHMAC := r.encryptor.HMAC(intentID)
	tag, err := r.db.Exec(ctx,
		`UPDATE payments
		 SET status = $2, error_code_enc = NULL, error_message_enc = NULL
		 WHERE stripe_payment_intent_id_hmac = $1`,
		piHMAC, StatusSucceeded,
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
	piHMAC := r.encryptor.HMAC(intentID)
	codeEnc, err := r.encryptor.Encrypt(code)
	if err != nil {
		return fmt.Errorf("encrypt error code: %w", err)
	}
	msgEnc, err := r.encryptor.Encrypt(message)
	if err != nil {
		return fmt.Errorf("encrypt error message: %w", err)
	}
	tag, err := r.db.Exec(ctx,
		`UPDATE payments
		 SET status = $2, error_code_enc = $3, error_message_enc = $4
		 WHERE stripe_payment_intent_id_hmac = $1`,
		piHMAC, StatusFailed, codeEnc, msgEnc,
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
	row := r.db.QueryRow(ctx,
		`SELECT `+selectColumns+`
		 FROM payments
		 WHERE order_id = $1
		 ORDER BY created_at DESC
		 LIMIT 1`,
		orderID,
	)
	p, err := r.scanRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("latest payment for order: %w", err)
	}
	return p, nil
}

// MarkRefunded flips the row to refunded, stamping the encrypted Stripe
// refund id and timestamp. Idempotent at the SQL level: a row already in
// 'refunded' is a no-op.
func (r *postgresRepository) MarkRefunded(ctx context.Context, paymentID uuid.UUID, refundID string) error {
	refundEnc, err := r.encryptor.Encrypt(refundID)
	if err != nil {
		return fmt.Errorf("encrypt refund id: %w", err)
	}
	tag, err := r.db.Exec(ctx,
		`UPDATE payments
		 SET status = $2, stripe_refund_id_enc = $3, refunded_at = NOW()
		 WHERE id = $1 AND status <> $2`,
		paymentID, StatusRefunded, refundEnc,
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
