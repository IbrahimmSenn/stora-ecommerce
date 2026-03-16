package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, token RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenString string) (*RefreshToken, error)
	MarkRefreshTokenUsed(ctx context.Context, tokenID string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
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

func (r *postgresAuthRepository) GetRefreshToken(ctx context.Context, tokenString string) (*RefreshToken, error) {
	query := `SELECT id, token, user_id, revoked, used, created_at, updated_at, expires_at
		FROM refresh_tokens WHERE token = $1`
	row := r.db.QueryRow(ctx, query, tokenString)

	var t RefreshToken
	err := row.Scan(&t.ID, &t.Token, &t.UserID, &t.Revoked, &t.Used, &t.CreatedAt, &t.UpdatedAt, &t.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &t, nil
}

func (r *postgresAuthRepository) MarkRefreshTokenUsed(ctx context.Context, tokenID string) error {
	query := `UPDATE refresh_tokens SET used = true WHERE id = $1`
	tag, err := r.db.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("mark refresh token used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTokenNotFound
	}
	return nil
}

func (r *postgresAuthRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	query := `UPDATE refresh_tokens SET revoked = true WHERE user_id = $1 AND revoked = false`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}
	return nil
}
