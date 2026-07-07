// repository.go — postgres queries for sitemap generation.
package seo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductEntry struct {
	ID        string
	UpdatedAt time.Time
}

type Repository interface {
	ProductEntries(ctx context.Context) ([]ProductEntry, error)
	CategorySlugs(ctx context.Context) ([]string, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) ProductEntries(ctx context.Context) ([]ProductEntry, error) {
	rows, err := r.db.Query(ctx, `SELECT id, updated_at FROM products ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list sitemap products: %w", err)
	}
	defer rows.Close()

	var entries []ProductEntry
	for rows.Next() {
		var e ProductEntry
		if err := rows.Scan(&e.ID, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sitemap product: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *postgresRepository) CategorySlugs(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT slug FROM categories ORDER BY slug`)
	if err != nil {
		return nil, fmt.Errorf("list sitemap categories: %w", err)
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan sitemap category: %w", err)
		}
		slugs = append(slugs, s)
	}
	return slugs, rows.Err()
}
