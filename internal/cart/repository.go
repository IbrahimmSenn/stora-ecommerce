package cart

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetOrCreateByUser(ctx context.Context, userID uuid.UUID) (*Cart, error)
	GetOrCreateByGuest(ctx context.Context, sessionID uuid.UUID) (*Cart, error)
	GetByUser(ctx context.Context, userID uuid.UUID) (*Cart, error)
	GetByGuest(ctx context.Context, sessionID uuid.UUID) (*Cart, error)
	GetByID(ctx context.Context, cartID uuid.UUID) (*Cart, error)
	AddItem(ctx context.Context, cartID, productID uuid.UUID, quantity int) (*CartItem, error)
	UpdateItemQuantity(ctx context.Context, cartID, productID uuid.UUID, quantity int) (*CartItem, error)
	RemoveItem(ctx context.Context, cartID, productID uuid.UUID) error
	ClearCart(ctx context.Context, cartID uuid.UUID) error
	DeleteCart(ctx context.Context, cartID uuid.UUID) error
	GetItems(ctx context.Context, cartID uuid.UUID) ([]CartItemDetail, error)
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetOrCreateByUser(ctx context.Context, userID uuid.UUID) (*Cart, error) {
	var c Cart
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, guest_session_id, created_at, updated_at
		 FROM carts WHERE user_id = $1`, userID,
	).Scan(&c.ID, &c.UserID, &c.GuestSessionID, &c.CreatedAt, &c.UpdatedAt)

	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get cart by user: %w", err)
	}

	err = r.db.QueryRow(ctx,
		`INSERT INTO carts (user_id) VALUES ($1)
		 RETURNING id, user_id, guest_session_id, created_at, updated_at`, userID,
	).Scan(&c.ID, &c.UserID, &c.GuestSessionID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create cart for user: %w", err)
	}
	return &c, nil
}

func (r *postgresRepository) GetOrCreateByGuest(ctx context.Context, sessionID uuid.UUID) (*Cart, error) {
	var c Cart
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, guest_session_id, created_at, updated_at
		 FROM carts WHERE guest_session_id = $1`, sessionID,
	).Scan(&c.ID, &c.UserID, &c.GuestSessionID, &c.CreatedAt, &c.UpdatedAt)

	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get cart by guest: %w", err)
	}

	err = r.db.QueryRow(ctx,
		`INSERT INTO carts (guest_session_id) VALUES ($1)
		 RETURNING id, user_id, guest_session_id, created_at, updated_at`, sessionID,
	).Scan(&c.ID, &c.UserID, &c.GuestSessionID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create cart for guest: %w", err)
	}
	return &c, nil
}

func (r *postgresRepository) GetByUser(ctx context.Context, userID uuid.UUID) (*Cart, error) {
	var c Cart
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, guest_session_id, created_at, updated_at
		 FROM carts WHERE user_id = $1`, userID,
	).Scan(&c.ID, &c.UserID, &c.GuestSessionID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("get cart by user: %w", err)
	}
	return &c, nil
}

func (r *postgresRepository) GetByGuest(ctx context.Context, sessionID uuid.UUID) (*Cart, error) {
	var c Cart
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, guest_session_id, created_at, updated_at
		 FROM carts WHERE guest_session_id = $1`, sessionID,
	).Scan(&c.ID, &c.UserID, &c.GuestSessionID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("get cart by guest: %w", err)
	}
	return &c, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, cartID uuid.UUID) (*Cart, error) {
	var c Cart
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, guest_session_id, created_at, updated_at
		 FROM carts WHERE id = $1`, cartID,
	).Scan(&c.ID, &c.UserID, &c.GuestSessionID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("get cart: %w", err)
	}
	return &c, nil
}

// AddItem inserts a new item or bumps quantity if the product already exists in the cart.
func (r *postgresRepository) AddItem(ctx context.Context, cartID, productID uuid.UUID, quantity int) (*CartItem, error) {
	var item CartItem
	err := r.db.QueryRow(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (cart_id, product_id)
		 DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity
		 RETURNING id, cart_id, product_id, quantity, created_at, updated_at`,
		cartID, productID, quantity,
	).Scan(&item.ID, &item.CartID, &item.ProductID, &item.Quantity, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("add cart item: %w", err)
	}
	return &item, nil
}

// UpdateItemQuantity atomically re-checks stock under SELECT ... FOR UPDATE on
// the product row, then writes the new quantity in the same transaction. Two
// concurrent calls for the same product can no longer each pass a stale read
// and write a combined quantity that exceeds stock.
func (r *postgresRepository) UpdateItemQuantity(ctx context.Context, cartID, productID uuid.UUID, quantity int) (*CartItem, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var stock int
	err = tx.QueryRow(ctx,
		`SELECT stock_quantity FROM products WHERE id = $1 FOR UPDATE`,
		productID,
	).Scan(&stock)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("lock product for cart update: %w", err)
	}
	if stock < quantity {
		return nil, ErrOutOfStock
	}

	var item CartItem
	err = tx.QueryRow(ctx,
		`UPDATE cart_items SET quantity = $3
		 WHERE cart_id = $1 AND product_id = $2
		 RETURNING id, cart_id, product_id, quantity, created_at, updated_at`,
		cartID, productID, quantity,
	).Scan(&item.ID, &item.CartID, &item.ProductID, &item.Quantity, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("update cart item: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit cart update tx: %w", err)
	}
	return &item, nil
}

func (r *postgresRepository) RemoveItem(ctx context.Context, cartID, productID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM cart_items WHERE cart_id = $1 AND product_id = $2`,
		cartID, productID,
	)
	if err != nil {
		return fmt.Errorf("remove cart item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrItemNotFound
	}
	return nil
}

func (r *postgresRepository) ClearCart(ctx context.Context, cartID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	if err != nil {
		return fmt.Errorf("clear cart: %w", err)
	}
	return nil
}

// DeleteCart removes the cart row itself; cart_items cascade via FK.
func (r *postgresRepository) DeleteCart(ctx context.Context, cartID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM carts WHERE id = $1`, cartID)
	if err != nil {
		return fmt.Errorf("delete cart: %w", err)
	}
	return nil
}

// GetItems returns all items in the cart joined with product info for display.
func (r *postgresRepository) GetItems(ctx context.Context, cartID uuid.UUID) ([]CartItemDetail, error) {
	rows, err := r.db.Query(ctx,
		`SELECT ci.id, ci.product_id, p.name, COALESCE(p.sale_price, p.price),
			(SELECT pi.url FROM product_images pi
			 WHERE pi.product_id = p.id AND pi.is_primary = true LIMIT 1),
			ci.quantity, p.stock_quantity
		 FROM cart_items ci
		 JOIN products p ON p.id = ci.product_id
		 WHERE ci.cart_id = $1
		 ORDER BY ci.created_at`, cartID,
	)
	if err != nil {
		return nil, fmt.Errorf("get cart items: %w", err)
	}
	defer rows.Close()

	var items []CartItemDetail
	for rows.Next() {
		var item CartItemDetail
		if err := rows.Scan(
			&item.ID, &item.ProductID, &item.ProductName, &item.ProductPrice,
			&item.ImageURL, &item.Quantity, &item.Stock,
		); err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []CartItemDetail{}
	}
	return items, nil
}
