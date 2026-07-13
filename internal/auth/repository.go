// repository.go — postgres queries for refresh tokens, password reset tokens, and 2FA records.
package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/crypto"
)

type AuthRepository interface {
	StoreRefreshToken(ctx context.Context, token RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenString string) (*RefreshToken, error)
	MarkRefreshTokenUsed(ctx context.Context, tokenID string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error

	// Password reset
	StorePasswordResetToken(ctx context.Context, token PasswordResetToken) error
	GetPasswordResetToken(ctx context.Context, tokenString string) (*PasswordResetToken, error)
	MarkResetTokenUsed(ctx context.Context, tokenID string) error

	// 2FA
	Store2FA(ctx context.Context, tfa TwoFactorAuth) error
	Get2FAByUserID(ctx context.Context, userID string) (*TwoFactorAuth, error)
	Enable2FA(ctx context.Context, userID string) error
	Delete2FA(ctx context.Context, userID string) error
	StoreRecoveryCodes(ctx context.Context, userID string, codes []string) error
}

type postgresAuthRepository struct {
	db  *pgxpool.Pool
	enc *crypto.Encryptor
}

func NewAuthRepository(db *pgxpool.Pool, enc *crypto.Encryptor) AuthRepository {
	return &postgresAuthRepository{db: db, enc: enc}
}

// encField encrypts a value and hex-encodes the ciphertext so it fits the
// existing TEXT columns (no schema change). decField reverses it.
func (r *postgresAuthRepository) encField(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	ct, err := r.enc.Encrypt(v)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ct), nil
}

func (r *postgresAuthRepository) decField(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	ct, err := hex.DecodeString(v)
	if err != nil {
		return "", fmt.Errorf("decode field: %w", err)
	}
	return r.enc.Decrypt(ct)
}

// --- Refresh tokens ---

func (r *postgresAuthRepository) StoreRefreshToken(ctx context.Context, token RefreshToken) error {
	// Store only the digest — never the raw token (see hash.go).
	query := `INSERT INTO refresh_tokens (id, token, user_id, expires_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(ctx, query, token.ID, hashToken(token.Token), token.UserID, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

func (r *postgresAuthRepository) GetRefreshToken(ctx context.Context, tokenString string) (*RefreshToken, error) {
	query := `SELECT id, token, user_id, revoked, used, created_at, updated_at, expires_at
		FROM refresh_tokens WHERE token = $1`
	row := r.db.QueryRow(ctx, query, hashToken(tokenString))

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

// MarkRefreshTokenUsed atomically flips used false->true. The AND used = false
// guard makes rotation race-safe: if two concurrent refreshes present the same
// token, only one UPDATE affects a row. The loser gets ErrTokenUsed, which the
// service treats as a replay (revokes the family). Zero rows with no matching
// unused token is reported as ErrTokenUsed rather than NotFound because the row
// existed at read time — it was just consumed by the winner.
func (r *postgresAuthRepository) MarkRefreshTokenUsed(ctx context.Context, tokenID string) error {
	query := `UPDATE refresh_tokens SET used = true WHERE id = $1 AND used = false`
	tag, err := r.db.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("mark refresh token used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTokenUsed
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

// --- Password reset tokens ---

func (r *postgresAuthRepository) StorePasswordResetToken(ctx context.Context, token PasswordResetToken) error {
	query := `INSERT INTO password_reset_tokens (id, user_id, token, expires_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(ctx, query, token.ID, token.UserID, hashToken(token.Token), token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}
	return nil
}

func (r *postgresAuthRepository) GetPasswordResetToken(ctx context.Context, tokenString string) (*PasswordResetToken, error) {
	query := `SELECT id, user_id, token, used, expires_at, created_at FROM password_reset_tokens WHERE token = $1`
	row := r.db.QueryRow(ctx, query, hashToken(tokenString))

	var t PasswordResetToken
	err := row.Scan(&t.ID, &t.UserID, &t.Token, &t.Used, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrResetTokenNotFound
		}
		return nil, fmt.Errorf("get reset token: %w", err)
	}
	return &t, nil
}

func (r *postgresAuthRepository) MarkResetTokenUsed(ctx context.Context, tokenID string) error {
	query := `UPDATE password_reset_tokens SET used = true WHERE id = $1`
	tag, err := r.db.Exec(ctx, query, tokenID)
	if err != nil {
		return fmt.Errorf("mark reset token used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrResetTokenNotFound
	}
	return nil
}

// --- Two-factor auth ---

func (r *postgresAuthRepository) Store2FA(ctx context.Context, tfa TwoFactorAuth) error {
	secret, err := r.encField(tfa.SecretKey)
	if err != nil {
		return fmt.Errorf("encrypt 2fa secret: %w", err)
	}
	query := `INSERT INTO two_factor_auth (id, user_id, secret_key) VALUES ($1, $2, $3)`
	_, err = r.db.Exec(ctx, query, tfa.ID, tfa.UserID, secret)
	if err != nil {
		return fmt.Errorf("store 2fa: %w", err)
	}
	return nil
}

func (r *postgresAuthRepository) Get2FAByUserID(ctx context.Context, userID string) (*TwoFactorAuth, error) {
	query := `SELECT id, user_id, secret_key, is_enabled, recovery_codes, created_at FROM two_factor_auth WHERE user_id = $1`
	row := r.db.QueryRow(ctx, query, userID)

	var tfa TwoFactorAuth
	err := row.Scan(&tfa.ID, &tfa.UserID, &tfa.SecretKey, &tfa.IsEnabled, &tfa.RecoveryCodes, &tfa.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, Err2FANotEnabled
		}
		return nil, fmt.Errorf("get 2fa: %w", err)
	}

	if tfa.SecretKey, err = r.decField(tfa.SecretKey); err != nil {
		return nil, fmt.Errorf("decrypt 2fa secret: %w", err)
	}
	for i, c := range tfa.RecoveryCodes {
		if tfa.RecoveryCodes[i], err = r.decField(c); err != nil {
			return nil, fmt.Errorf("decrypt recovery code: %w", err)
		}
	}
	return &tfa, nil
}

func (r *postgresAuthRepository) Enable2FA(ctx context.Context, userID string) error {
	query := `UPDATE two_factor_auth SET is_enabled = true WHERE user_id = $1`
	tag, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("enable 2fa: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Err2FANotEnabled
	}
	return nil
}

func (r *postgresAuthRepository) Delete2FA(ctx context.Context, userID string) error {
	query := `DELETE FROM two_factor_auth WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete 2fa: %w", err)
	}
	return nil
}

func (r *postgresAuthRepository) StoreRecoveryCodes(ctx context.Context, userID string, codes []string) error {
	enc := make([]string, len(codes))
	for i, c := range codes {
		v, err := r.encField(c)
		if err != nil {
			return fmt.Errorf("encrypt recovery code: %w", err)
		}
		enc[i] = v
	}
	query := `UPDATE two_factor_auth SET recovery_codes = $1 WHERE user_id = $2`
	_, err := r.db.Exec(ctx, query, enc, userID)
	if err != nil {
		return fmt.Errorf("store recovery codes: %w", err)
	}
	return nil
}
