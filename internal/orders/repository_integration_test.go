//go:build integration

// These tests exercise the real Postgres row-locking and compare-and-set paths
// that unit tests with in-memory stubs cannot reach. They run only under
// `-tags=integration` against TEST_DATABASE_URL (a migrated schema).
package orders

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/testdb"
)

var errOutOfStock = errors.New("itest: out of stock")

func insertProduct(t *testing.T, pool *pgxpool.Pool, stock int) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO products (id, name, price, stock_quantity, weight_g) VALUES ($1,$2,$3,$4,$5)`,
		id, "itest-product", 1000, stock, 100)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, id) })
	return id
}

func insertOrder(t *testing.T, pool *pgxpool.Pool, status string, productID uuid.UUID, qty int) uuid.UUID {
	t.Helper()
	orderID := uuid.New()
	guest := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO orders (id, order_number, guest_session_id, status, email_encrypted,
			subtotal_cents, shipping_cents, total_cents, shipping_method)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		orderID, "ITEST-"+orderID.String()[:8], guest, status, []byte("x"),
		1000, 0, 1000, "standard")
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		`INSERT INTO order_items (order_id, product_id, product_name, unit_price_cents, quantity)
		 VALUES ($1,$2,$3,$4,$5)`,
		orderID, productID, "itest-product", 1000, qty)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID) })
	return orderID
}

func stockOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT stock_quantity FROM products WHERE id = $1`, id).Scan(&n))
	return n
}

// TestIntegration_OversellGuard: many buyers race for the last unit. The
// SELECT ... FOR UPDATE serializes them, so exactly one decrement succeeds and
// stock never goes negative.
func TestIntegration_OversellGuard(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepository(pool)
	productID := insertProduct(t, pool, 1)

	const buyers = 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0

	for i := 0; i < buyers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.WithTx(context.Background(), func(tx TxRepo) error {
				p, err := tx.LockProductForUpdate(context.Background(), productID)
				if err != nil {
					return err
				}
				if p.Stock < 1 {
					return errOutOfStock
				}
				return tx.DecrementStock(context.Background(), productID, 1)
			})
			if err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, successes, "only one buyer should win the last unit")
	assert.Equal(t, 0, stockOf(t, pool, productID), "stock must not go negative")
}

// TestIntegration_ConcurrentRefundRestocksOnce: two refunds race on the same
// paid order. The TransitionStatus compare-and-set lets only one flip
// paid->refunded, so the restock runs exactly once.
func TestIntegration_ConcurrentRefundRestocksOnce(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepository(pool)
	productID := insertProduct(t, pool, 0)
	orderID := insertOrder(t, pool, StatusPaid, productID, 3)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = repo.WithTx(context.Background(), func(tx TxRepo) error {
				ok, err := tx.TransitionStatus(context.Background(), orderID, StatusPaid, StatusRefunded)
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
				return tx.IncrementStock(context.Background(), productID, 3)
			})
		}()
	}
	wg.Wait()

	assert.Equal(t, 3, stockOf(t, pool, productID), "restock must apply exactly once")
}
