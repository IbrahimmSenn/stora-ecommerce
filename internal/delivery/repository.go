// repository.go — postgres queries for delivery options.
package delivery

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolation = "23505"

type Repository interface {
	List(ctx context.Context, activeOnly bool) ([]DeliveryOption, error)
	Create(ctx context.Context, o DeliveryOption) (*DeliveryOption, error)
	Update(ctx context.Context, id string, o DeliveryOption) (*DeliveryOption, error)
	Delete(ctx context.Context, id string) error
	// RateByCode returns the price for an active option with the given code.
	// ok is false when no active option matches.
	RateByCode(ctx context.Context, code string) (cents int64, ok bool, err error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

const cols = `id, code, label, price_cents, eta_label, sort_order, active, created_at, updated_at`

func scanOption(row pgx.Row) (*DeliveryOption, error) {
	var o DeliveryOption
	if err := row.Scan(&o.ID, &o.Code, &o.Label, &o.PriceCents, &o.EtaLabel,
		&o.SortOrder, &o.Active, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *postgresRepository) List(ctx context.Context, activeOnly bool) ([]DeliveryOption, error) {
	q := `SELECT ` + cols + ` FROM delivery_options`
	if activeOnly {
		q += ` WHERE active = true`
	}
	q += ` ORDER BY sort_order, price_cents`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list delivery options: %w", err)
	}
	defer rows.Close()

	out := []DeliveryOption{}
	for rows.Next() {
		o, err := scanOption(rows)
		if err != nil {
			return nil, fmt.Errorf("scan delivery option: %w", err)
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func (r *postgresRepository) Create(ctx context.Context, o DeliveryOption) (*DeliveryOption, error) {
	q := `INSERT INTO delivery_options (code, label, price_cents, eta_label, sort_order, active)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING ` + cols
	created, err := scanOption(r.db.QueryRow(ctx, q,
		o.Code, o.Label, o.PriceCents, o.EtaLabel, o.SortOrder, o.Active))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return nil, ErrCodeExists
		}
		return nil, fmt.Errorf("create delivery option: %w", err)
	}
	return created, nil
}

func (r *postgresRepository) Update(ctx context.Context, id string, o DeliveryOption) (*DeliveryOption, error) {
	q := `UPDATE delivery_options
		SET label = $2, price_cents = $3, eta_label = $4, sort_order = $5, active = $6, updated_at = NOW()
		WHERE id = $1 RETURNING ` + cols
	updated, err := scanOption(r.db.QueryRow(ctx, q,
		id, o.Label, o.PriceCents, o.EtaLabel, o.SortOrder, o.Active))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update delivery option: %w", err)
	}
	return updated, nil
}

func (r *postgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM delivery_options WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete delivery option: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) RateByCode(ctx context.Context, code string) (int64, bool, error) {
	var cents int64
	err := r.db.QueryRow(ctx,
		`SELECT price_cents FROM delivery_options WHERE code = $1 AND active = true`, code).Scan(&cents)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("rate by code: %w", err)
	}
	return cents, true, nil
}
