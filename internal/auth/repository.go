package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, token RefreshToken) error
}

type postgresAuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) AuthRepository {
	return &postgresAuthRepository{db: db}
}

func (r *postgresAuthRepository) StoreRefreshToken(ctx context.Context, token RefreshToken) error {
	query := `INSERT INTO refresh_tokens (id, token, user_id, expires_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(ctx, query, token.ID, token.Token, token.UserID, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}
