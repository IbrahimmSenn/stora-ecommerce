// repository.go — postgres queries for categories (supports hierarchical parent_id).
package category

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolation = "23505"

type Repository interface {
	List(ctx context.Context) ([]Category, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	GetBySlug(ctx context.Context, slug string) (*Category, error)
	Create(ctx context.Context, name, slug string, parentID *uuid.UUID) (*Category, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) List(ctx context.Context) ([]Category, error) {
	query := `SELECT id, name, slug, parent_id, created_at, updated_at FROM categories ORDER BY name`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*Category, error) {
	query := `SELECT id, name, slug, parent_id, created_at, updated_at FROM categories WHERE id = $1`
	var c Category
	err := r.db.QueryRow(ctx, query, id).Scan(&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("get category: %w", err)
	}
	return &c, nil
}

func (r *postgresRepository) GetBySlug(ctx context.Context, slug string) (*Category, error) {
	query := `SELECT id, name, slug, parent_id, created_at, updated_at FROM categories WHERE slug = $1`
	var c Category
	err := r.db.QueryRow(ctx, query, slug).Scan(&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("get category by slug: %w", err)
	}
	return &c, nil
}

func (r *postgresRepository) Create(ctx context.Context, name, slug string, parentID *uuid.UUID) (*Category, error) {
	query := `INSERT INTO categories (name, slug, parent_id) VALUES ($1, $2, $3)
		RETURNING id, name, slug, parent_id, created_at, updated_at`
	var c Category
	err := r.db.QueryRow(ctx, query, name, slug, parentID).Scan(
		&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return nil, ErrCategoryExists
		}
		return nil, fmt.Errorf("create category: %w", err)
	}
	return &c, nil
}
