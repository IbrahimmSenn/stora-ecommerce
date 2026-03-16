package oauth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetUserIDByProvider(ctx context.Context, provider, providerUserID string) (string, error)
	LinkAccount(ctx context.Context, userID, provider, providerUserID string) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetUserIDByProvider(ctx context.Context, provider, providerUserID string) (string, error) {
	query := `SELECT user_id FROM oauth_accounts WHERE provider = $1 AND provider_user_id = $2`
	var userID string
	err := r.db.QueryRow(ctx, query, provider, providerUserID).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("oauth account not found")
		}
		return "", fmt.Errorf("get oauth user: %w", err)
	}
	return userID, nil
}

func (r *postgresRepository) LinkAccount(ctx context.Context, userID, provider, providerUserID string) error {
	query := `INSERT INTO oauth_accounts (user_id, provider, provider_user_id) VALUES ($1, $2, $3)
		ON CONFLICT (provider, provider_user_id) DO NOTHING`
	_, err := r.db.Exec(ctx, query, userID, provider, providerUserID)
	if err != nil {
		return fmt.Errorf("link oauth account: %w", err)
	}
	return nil
}
