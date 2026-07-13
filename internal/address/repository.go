// repository.go — postgres queries for saved addresses. Address fields are
// AES-256-GCM encrypted on write and decrypted on read at this boundary.
package address

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/crypto"
)

type Repository interface {
	Create(ctx context.Context, userID string, req AddressRequest) (*Address, error)
	List(ctx context.Context, userID string) ([]Address, error)
	Update(ctx context.Context, userID, id string, req AddressRequest) (*Address, error)
	Delete(ctx context.Context, userID, id string) error
	SetDefault(ctx context.Context, userID, id string) error
}

type postgresRepository struct {
	db  *pgxpool.Pool
	enc *crypto.Encryptor
}

func NewRepository(db *pgxpool.Pool, enc *crypto.Encryptor) Repository {
	return &postgresRepository{db: db, enc: enc}
}

// encrypted holds the seven encrypted address fields.
type encrypted struct {
	recipient, line1, line2, city, region, postal, country []byte
}

func (r *postgresRepository) encryptReq(req AddressRequest) (*encrypted, error) {
	var e encrypted
	var err error
	put := func(dst *[]byte, v string) {
		if err != nil {
			return
		}
		*dst, err = r.enc.Encrypt(v)
	}
	put(&e.recipient, req.RecipientName)
	put(&e.line1, req.Line1)
	put(&e.line2, req.Line2)
	put(&e.city, req.City)
	put(&e.region, req.Region)
	put(&e.postal, req.PostalCode)
	put(&e.country, req.Country)
	if err != nil {
		return nil, fmt.Errorf("encrypt address: %w", err)
	}
	return &e, nil
}

func (r *postgresRepository) Create(ctx context.Context, userID string, req AddressRequest) (*Address, error) {
	e, err := r.encryptReq(req)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// First address for a user is always the default; an explicit default unsets
	// the others.
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM saved_addresses WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return nil, err
	}
	makeDefault := req.IsDefault || count == 0
	if makeDefault {
		if _, err := tx.Exec(ctx, `UPDATE saved_addresses SET is_default = false WHERE user_id = $1`, userID); err != nil {
			return nil, err
		}
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO saved_addresses
			(user_id, label, recipient_name_encrypted, line1_encrypted, line2_encrypted,
			 city_encrypted, region_encrypted, postal_code_encrypted, country_encrypted, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
		userID, nullableLabel(req.Label), e.recipient, e.line1, e.line2, e.city, e.region, e.postal, e.country, makeDefault,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert address: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.getOne(ctx, userID, id)
}

func (r *postgresRepository) Update(ctx context.Context, userID, id string, req AddressRequest) (*Address, error) {
	e, err := r.encryptReq(req)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if req.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE saved_addresses SET is_default = false WHERE user_id = $1`, userID); err != nil {
			return nil, err
		}
	}

	tag, err := tx.Exec(ctx, `
		UPDATE saved_addresses SET
			label = $3, recipient_name_encrypted = $4, line1_encrypted = $5, line2_encrypted = $6,
			city_encrypted = $7, region_encrypted = $8, postal_code_encrypted = $9, country_encrypted = $10,
			is_default = is_default OR $11
		WHERE id = $1 AND user_id = $2`,
		id, userID, nullableLabel(req.Label), e.recipient, e.line1, e.line2, e.city, e.region, e.postal, e.country, req.IsDefault)
	if err != nil {
		return nil, fmt.Errorf("update address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.getOne(ctx, userID, id)
}

func (r *postgresRepository) SetDefault(ctx context.Context, userID, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE saved_addresses SET is_default = false WHERE user_id = $1`, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE saved_addresses SET is_default = true WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (r *postgresRepository) Delete(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM saved_addresses WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) List(ctx context.Context, userID string) ([]Address, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, label, recipient_name_encrypted, line1_encrypted, line2_encrypted,
		       city_encrypted, region_encrypted, postal_code_encrypted, country_encrypted,
		       is_default, created_at
		FROM saved_addresses WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	defer rows.Close()

	out := []Address{}
	for rows.Next() {
		a, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate addresses: %w", err)
	}
	return out, nil
}

func (r *postgresRepository) getOne(ctx context.Context, userID, id string) (*Address, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, label, recipient_name_encrypted, line1_encrypted, line2_encrypted,
		       city_encrypted, region_encrypted, postal_code_encrypted, country_encrypted,
		       is_default, created_at
		FROM saved_addresses WHERE id = $1 AND user_id = $2`, id, userID)
	a, err := r.scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *postgresRepository) scan(row pgx.Row) (*Address, error) {
	var a Address
	var label *string
	var rec, l1, l2, city, region, postal, country []byte
	if err := row.Scan(&a.ID, &label, &rec, &l1, &l2, &city, &region, &postal, &country, &a.IsDefault, &a.CreatedAt); err != nil {
		return nil, err
	}
	a.Label = label

	dec := func(b []byte) string {
		s, derr := r.enc.Decrypt(b)
		if derr != nil {
			return ""
		}
		return s
	}
	a.RecipientName = dec(rec)
	a.Line1 = dec(l1)
	a.Line2 = dec(l2)
	a.City = dec(city)
	a.Region = dec(region)
	a.PostalCode = dec(postal)
	a.Country = dec(country)
	return &a, nil
}

func nullableLabel(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
