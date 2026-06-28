// repository.go — postgres queries for user CRUD. Email is encrypted at rest
// (email_encrypted) with a deterministic HMAC blind index (email_hmac) for
// equality lookup at login and uniqueness. Email is normalised (lower-cased,
// trimmed) before hashing/encrypting so the blind index is stable regardless of
// how a caller cased it.
package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/crypto"
)

const uniqueViolation = "23505" // PostgreSQL unique constraint violation code

type UserRepository interface {
	CreateUser(ctx context.Context, user User) error
	CreateOAuthUser(ctx context.Context, user User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	UpdatePassword(ctx context.Context, userID string, passwordHash string) error
	ListAll(ctx context.Context, limit, offset int) ([]User, int, error)
	UpdateRole(ctx context.Context, userID, role string) error
	CountByRole(ctx context.Context, role string) (int, error)
}

type postgresUserRepository struct {
	db  *pgxpool.Pool
	enc *crypto.Encryptor
}

func NewUserRepository(db *pgxpool.Pool, enc *crypto.Encryptor) UserRepository {
	return &postgresUserRepository{db: db, enc: enc}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (r *postgresUserRepository) CreateUser(ctx context.Context, user User) error {
	email := normalizeEmail(user.Email)
	enc, err := r.enc.Encrypt(email)
	if err != nil {
		return fmt.Errorf("encrypt email: %w", err)
	}
	query := `INSERT INTO users (email_encrypted, email_hmac, password_hash) VALUES ($1, $2, $3)`
	_, err = r.db.Exec(ctx, query, enc, r.enc.HMAC(email), user.PasswordHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return ErrEmailExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) CreateOAuthUser(ctx context.Context, user User) error {
	email := normalizeEmail(user.Email)
	enc, err := r.enc.Encrypt(email)
	if err != nil {
		return fmt.Errorf("encrypt email: %w", err)
	}
	query := `INSERT INTO users (id, email_encrypted, email_hmac, password_hash, role) VALUES ($1, $2, $3, '', $4)`
	_, err = r.db.Exec(ctx, query, user.Id, enc, r.enc.HMAC(email), user.Role)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return ErrEmailExists
		}
		return fmt.Errorf("create oauth user: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, email_encrypted, password_hash, role, created_at, updated_at FROM users WHERE email_hmac = $1`
	row := r.db.QueryRow(ctx, query, r.enc.HMAC(normalizeEmail(email)))
	return r.scanUser(row)
}

func (r *postgresUserRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, email_encrypted, password_hash, role, created_at, updated_at FROM users WHERE id = $1`
	return r.scanUser(r.db.QueryRow(ctx, query, id))
}

func (r *postgresUserRepository) scanUser(row pgx.Row) (*User, error) {
	var user User
	var emailEnc []byte
	err := row.Scan(&user.Id, &emailEnc, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.Email, err = r.enc.Decrypt(emailEnc); err != nil {
		return nil, fmt.Errorf("decrypt email: %w", err)
	}
	return &user, nil
}

func (r *postgresUserRepository) ListAll(ctx context.Context, limit, offset int) ([]User, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, email_encrypted, role, created_at, updated_at FROM users
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	out := []User{}
	for rows.Next() {
		var u User
		var emailEnc []byte
		if err := rows.Scan(&u.Id, &emailEnc, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		if u.Email, err = r.enc.Decrypt(emailEnc); err != nil {
			return nil, 0, fmt.Errorf("decrypt email: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate users: %w", err)
	}
	return out, total, nil
}

func (r *postgresUserRepository) UpdateRole(ctx context.Context, userID, role string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`, role, userID)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *postgresUserRepository) CountByRole(ctx context.Context, role string) (int, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = $1`, role).Scan(&n); err != nil {
		return 0, fmt.Errorf("count by role: %w", err)
	}
	return n, nil
}

func (r *postgresUserRepository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	tag, err := r.db.Exec(ctx, query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
