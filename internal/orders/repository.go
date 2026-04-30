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

type Repository interface {
	BeginTx(ctx context.Context) (pgx.Tx, error)

	// Tx-scoped helpers used by the checkout flow. They live here (rather than
	// on cart/product repos) so the orders service can drive a single
	// transaction across orders + stock decrement + cart clear.
	LockProductForUpdateTx(ctx context.Context, tx pgx.Tx, productID uuid.UUID) (*LockedProduct, error)
	DecrementStockTx(ctx context.Context, tx pgx.Tx, productID uuid.UUID, quantity int) error
	IncrementStockTx(ctx context.Context, tx pgx.Tx, productID uuid.UUID, quantity int) error
	DeleteCartItemsTx(ctx context.Context, tx pgx.Tx, cartID uuid.UUID) error

	CreateOrderTx(ctx context.Context, tx pgx.Tx, row *orderRow) error
	CreateOrderItemTx(ctx context.Context, tx pgx.Tx, item *OrderItem) error
	CreateShippingAddressTx(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, addr *addressRow) error

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

func (r *postgresRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	return tx, nil
}

func (r *postgresRepository) LockProductForUpdateTx(ctx context.Context, tx pgx.Tx, productID uuid.UUID) (*LockedProduct, error) {
	var p LockedProduct
	err := tx.QueryRow(ctx,
		`SELECT id, name, price, stock_quantity FROM products WHERE id = $1 FOR UPDATE`,
		productID,
	).Scan(&p.ID, &p.Name, &p.Price, &p.Stock)
	if err != nil {
		return nil, fmt.Errorf("lock product: %w", err)
	}
	return &p, nil
}

func (r *postgresRepository) DecrementStockTx(ctx context.Context, tx pgx.Tx, productID uuid.UUID, quantity int) error {
	tag, err := tx.Exec(ctx,
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

func (r *postgresRepository) IncrementStockTx(ctx context.Context, tx pgx.Tx, productID uuid.UUID, quantity int) error {
	_, err := tx.Exec(ctx,
		`UPDATE products SET stock_quantity = stock_quantity + $2 WHERE id = $1`,
		productID, quantity,
	)
	if err != nil {
		return fmt.Errorf("increment stock: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeleteCartItemsTx(ctx context.Context, tx pgx.Tx, cartID uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	if err != nil {
		return fmt.Errorf("clear cart items: %w", err)
	}
	return nil
}

func (r *postgresRepository) CreateOrderTx(ctx context.Context, tx pgx.Tx, row *orderRow) error {
	number, err := generateOrderNumber()
	if err != nil {
		return fmt.Errorf("generate order number: %w", err)
	}
	row.OrderNumber = number

	err = tx.QueryRow(ctx,
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

func (r *postgresRepository) CreateOrderItemTx(ctx context.Context, tx pgx.Tx, item *OrderItem) error {
	err := tx.QueryRow(ctx,
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

func (r *postgresRepository) CreateShippingAddressTx(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, a *addressRow) error {
	_, err := tx.Exec(ctx,
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

// ItemsForRestock returns the items needed to restore stock when an order is
// cancelled. Filters out items whose product was deleted (product_id NULL).
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
