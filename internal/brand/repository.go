// repository.go — postgres queries for brands.
package brand

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
	List(ctx context.Context) ([]Brand, error)
	GetByID(ctx context.Context, id string) (*Brand, error)
	Create(ctx context.Context, name string) (*Brand, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) List(ctx context.Context) ([]Brand, error) {
	query := `SELECT id, name, created_at, updated_at FROM brands ORDER BY name`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list brands: %w", err)
	}
	defer rows.Close()

	var brands []Brand
	for rows.Next() {
		var b Brand
		if err := rows.Scan(&b.ID, &b.Name, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan brand: %w", err)
		}
		brands = append(brands, b)
	}
	return brands, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*Brand, error) {
	query := `SELECT id, name, created_at, updated_at FROM brands WHERE id = $1`
	var b Brand
	err := r.db.QueryRow(ctx, query, id).Scan(&b.ID, &b.Name, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBrandNotFound
		}
		return nil, fmt.Errorf("get brand: %w", err)
	}
	return &b, nil
}

func (r *postgresRepository) Create(ctx context.Context, name string) (*Brand, error) {
	query := `INSERT INTO brands (name) VALUES ($1) RETURNING id, name, created_at, updated_at`
	var b Brand
	err := r.db.QueryRow(ctx, query, name).Scan(&b.ID, &b.Name, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return nil, ErrBrandExists
		}
		return nil, fmt.Errorf("create brand: %w", err)
	}
	return &b, nil
}
