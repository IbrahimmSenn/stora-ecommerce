package orders

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LockedProduct is the snapshot of a product row taken under SELECT ... FOR
// UPDATE, used by the service to re-validate stock and price during checkout.
type LockedProduct struct {
	ID    uuid.UUID
	Name  string
	Price int64
	Stock int
}

// TxRepo is the set of operations the service runs inside a single
// transaction during checkout and cancel. The repository owns transaction
// lifecycle (see Repository.WithTx) so the service never touches pgx.Tx
// directly — that keeps unit tests free of pgx-mocking gymnastics.
type TxRepo interface {
	LockProductForUpdate(ctx context.Context, productID uuid.UUID) (*LockedProduct, error)
	DecrementStock(ctx context.Context, productID uuid.UUID, quantity int) error
	IncrementStock(ctx context.Context, productID uuid.UUID, quantity int) error
	DeleteCartItems(ctx context.Context, cartID uuid.UUID) error
	CreateOrder(ctx context.Context, row *orderRow) error
	CreateOrderItem(ctx context.Context, item *OrderItem) error
	CreateShippingAddress(ctx context.Context, orderID uuid.UUID, addr *addressRow) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}

type Repository interface {
	WithTx(ctx context.Context, fn func(tx TxRepo) error) error

	GetByID(ctx context.Context, id uuid.UUID) (*orderRow, []OrderItem, *addressRow, error)
	ListByUser(ctx context.Context, userID uuid.UUID, status string, from, to *time.Time) ([]OrderSummary, error)
	ListByGuest(ctx context.Context, guestSessionID uuid.UUID, status string, from, to *time.Time) ([]OrderSummary, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	ItemsForRestock(ctx context.Context, orderID uuid.UUID) ([]OrderItem, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) WithTx(ctx context.Context, fn func(tx TxRepo) error) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(&pgTx{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*orderRow, []OrderItem, *addressRow, error) {
	var o orderRow
	err := r.db.QueryRow(ctx,
		`SELECT id, order_number, user_id, guest_session_id, status,
			email_encrypted, phone_encrypted,
			subtotal_cents, shipping_cents, total_cents, shipping_method,
			created_at, updated_at
		 FROM orders WHERE id = $1`, id,
	).Scan(
		&o.ID, &o.OrderNumber, &o.UserID, &o.GuestSessionID, &o.Status,
		&o.EmailEnc, &o.PhoneEnc,
		&o.SubtotalCents, &o.ShippingCents, &o.TotalCents, &o.ShippingMethod,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, ErrOrderNotFound
		}
		return nil, nil, nil, fmt.Errorf("get order: %w", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, order_id, product_id, product_name, unit_price_cents, quantity, created_at
		 FROM order_items WHERE order_id = $1 ORDER BY created_at`, id,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query order items: %w", err)
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(
			&it.ID, &it.OrderID, &it.ProductID, &it.ProductName,
			&it.UnitPriceCents, &it.Quantity, &it.CreatedAt,
		); err != nil {
			return nil, nil, nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, it)
	}

	var a addressRow
	err = r.db.QueryRow(ctx,
		`SELECT recipient_name_encrypted, line1_encrypted, line2_encrypted,
			city_encrypted, region_encrypted, postal_code_encrypted, country_encrypted
		 FROM shipping_addresses WHERE order_id = $1`, id,
	).Scan(
		&a.RecipientNameEnc, &a.Line1Enc, &a.Line2Enc,
		&a.CityEnc, &a.RegionEnc, &a.PostalCodeEnc, &a.CountryEnc,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get shipping address: %w", err)
	}

	return &o, items, &a, nil
}

func (r *postgresRepository) ListByUser(ctx context.Context, userID uuid.UUID, status string, from, to *time.Time) ([]OrderSummary, error) {
	return r.listSummaries(ctx, "user_id", userID, status, from, to)
}

func (r *postgresRepository) ListByGuest(ctx context.Context, guestSessionID uuid.UUID, status string, from, to *time.Time) ([]OrderSummary, error) {
	return r.listSummaries(ctx, "guest_session_id", guestSessionID, status, from, to)
}

// listSummaries shares the SELECT between user/guest. ownerCol is hard-coded
// to one of two literals — never user input — so string interpolation is safe.
func (r *postgresRepository) listSummaries(ctx context.Context, ownerCol string, ownerID uuid.UUID, status string, from, to *time.Time) ([]OrderSummary, error) {
	if ownerCol != "user_id" && ownerCol != "guest_session_id" {
		return nil, fmt.Errorf("invalid owner column %q", ownerCol)
	}

	q := `SELECT o.id, o.order_number, o.status, o.total_cents,
			(SELECT COUNT(*) FROM order_items WHERE order_id = o.id) AS item_count,
			o.created_at
		 FROM orders o
		 WHERE o.` + ownerCol + ` = $1
			AND ($2::text IS NULL OR o.status = $2)
			AND ($3::timestamptz IS NULL OR o.created_at >= $3)
			AND ($4::timestamptz IS NULL OR o.created_at <= $4)
		 ORDER BY o.created_at DESC`

	var statusArg any
	if status != "" {
		statusArg = status
	}

	rows, err := r.db.Query(ctx, q, ownerID, statusArg, from, to)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	out := []OrderSummary{}
	for rows.Next() {
		var s OrderSummary
		if err := rows.Scan(&s.ID, &s.OrderNumber, &s.Status, &s.TotalCents, &s.ItemCount, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan order summary: %w", err)
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *postgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := r.db.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (r *postgresRepository) ItemsForRestock(ctx context.Context, orderID uuid.UUID) ([]OrderItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, order_id, product_id, product_name, unit_price_cents, quantity, created_at
		 FROM order_items WHERE order_id = $1 AND product_id IS NOT NULL`, orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("query items for restock: %w", err)
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(
			&it.ID, &it.OrderID, &it.ProductID, &it.ProductName,
			&it.UnitPriceCents, &it.Quantity, &it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, it)
	}
	return items, nil
}

// pgTx is the production TxRepo backed by a real pgx.Tx.

type pgTx struct {
	tx pgx.Tx
}

func (t *pgTx) LockProductForUpdate(ctx context.Context, productID uuid.UUID) (*LockedProduct, error) {
	var p LockedProduct
	err := t.tx.QueryRow(ctx,
		`SELECT id, name, price, stock_quantity FROM products WHERE id = $1 FOR UPDATE`,
		productID,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Stock)
	if err != nil {
		return nil, fmt.Errorf("lock product: %w", err)
	}
	return &p, nil
}

func (t *pgTx) DecrementStock(ctx context.Context, productID uuid.UUID, quantity int) error {
	tag, err := t.tx.Exec(ctx,
		`UPDATE products SET stock_quantity = stock_quantity - $2 WHERE id = $1`,
		productID, quantity,
	)
	if err != nil {
		return fmt.Errorf("decrement stock: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("decrement stock: product %s missing", productID)
	}
	return nil
}

func (t *pgTx) IncrementStock(ctx context.Context, productID uuid.UUID, quantity int) error {
	_, err := t.tx.Exec(ctx,
		`UPDATE products SET stock_quantity = stock_quantity + $2 WHERE id = $1`,
		productID, quantity,
	)
	if err != nil {
		return fmt.Errorf("increment stock: %w", err)
	}
	return nil
}

func (t *pgTx) DeleteCartItems(ctx context.Context, cartID uuid.UUID) error {
	_, err := t.tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	if err != nil {
		return fmt.Errorf("clear cart items: %w", err)
	}
	return nil
}

func (t *pgTx) CreateOrder(ctx context.Context, row *orderRow) error {
	number, err := generateOrderNumber()
	if err != nil {
		return fmt.Errorf("generate order number: %w", err)
	}
	row.OrderNumber = number

	err = t.tx.QueryRow(ctx,
		`INSERT INTO orders (
			order_number, user_id, guest_session_id, status,
			email_encrypted, phone_encrypted,
			subtotal_cents, shipping_cents, total_cents, shipping_method
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at, updated_at`,
		row.OrderNumber, row.UserID, row.GuestSessionID, row.Status,
		row.EmailEnc, row.PhoneEnc,
		row.SubtotalCents, row.ShippingCents, row.TotalCents, row.ShippingMethod,
	).Scan(&row.ID, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (t *pgTx) CreateOrderItem(ctx context.Context, item *OrderItem) error {
	err := t.tx.QueryRow(ctx,
		`INSERT INTO order_items (order_id, product_id, product_name, unit_price_cents, quantity)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, created_at`,
		item.OrderID, item.ProductID, item.ProductName, item.UnitPriceCents, item.Quantity,
	).Scan(&item.ID, &item.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert order item: %w", err)
	}
	return nil
}

func (t *pgTx) CreateShippingAddress(ctx context.Context, orderID uuid.UUID, a *addressRow) error {
	_, err := t.tx.Exec(ctx,
		`INSERT INTO shipping_addresses (
			order_id,
			recipient_name_encrypted, line1_encrypted, line2_encrypted,
			city_encrypted, region_encrypted, postal_code_encrypted, country_encrypted
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		orderID,
		a.RecipientNameEnc, a.Line1Enc, a.Line2Enc,
		a.CityEnc, a.RegionEnc, a.PostalCodeEnc, a.CountryEnc,
	)
	if err != nil {
		return fmt.Errorf("insert shipping address: %w", err)
	}
	return nil
}

func (t *pgTx) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := t.tx.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

// generateOrderNumber returns a 17-char human-friendly id like ORD-K7H4Z2QF8M3.
// 8 random bytes -> 13 base32 chars, prefixed with ORD-. Collision odds are
// negligible at any plausible volume; the orders.order_number unique index
// catches the rare retry case.
func generateOrderNumber() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "ORD-" + base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:]), nil
}
