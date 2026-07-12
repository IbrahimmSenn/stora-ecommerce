// history.go — backdated demo data for the observability dashboards.
//
// The regular seed (seed.sql + Demo) creates a catalogue and three users but
// no orders, and every timestamp is "now" — useless for time-series panels.
// History writes ~60 days of users, orders, abandoned carts, reviews, and
// funnel activity (views/add-to-carts/purchases) with explicit created_at
// values via direct SQL (the repositories deliberately can't backdate rows).
//
// Deterministic and idempotent: fixed RNG seed, fixed UUIDs, ON CONFLICT DO
// NOTHING — safe to re-run after a `make reset`.
//
// Two patterns are engineered on purpose so the correlated dashboard panels
// have something to say:
//   - registered customers get more items per order than guests (AOV gap)
//   - the two highest-revenue categories get mediocre ratings while a
//     low-revenue category gets excellent ones (rating/revenue divergence)
package seed

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/crypto"
)

const (
	historyDays     = 60
	historyUsers    = 300
	historyOrders   = 900
	historyCarts    = 150
	historyReviews  = 500
	historySessions = 800 // browse-only sessions that never convert
	historyPassword = "seed-history-password"

	// Sentinel guest session marking that activity history exists —
	// user_activity has no natural key for ON CONFLICT, so re-runs check
	// for this row instead.
	activitySentinel = "a0000000-0000-4000-8000-000000000000"
)

type histProduct struct {
	id         string
	price      int64
	categoryID string
}

// History seeds the backdated dataset. Counts are reported so the CLI can
// print a summary.
func History(ctx context.Context, db *pgxpool.Pool, enc *crypto.Encryptor) error {
	rng := rand.New(rand.NewSource(42)) // #nosec G404 -- deterministic demo data, not security

	products, categories, err := loadCatalog(ctx, db)
	if err != nil {
		return err
	}
	if len(products) == 0 {
		return fmt.Errorf("no products found — run migrations + seed.sql first")
	}

	// Category weighting: the first two (sorted) categories carry most of the
	// revenue, the last one barely sells but reviews brilliantly.
	heavy := map[string]bool{}
	var lowRev string
	if len(categories) >= 3 {
		heavy[categories[0]] = true
		heavy[categories[1]] = true
		lowRev = categories[len(categories)-1]
	}

	now := time.Now()
	start := now.AddDate(0, 0, -historyDays)

	hash, err := bcrypt.GenerateFromPassword([]byte(historyPassword), bcrypt.MinCost)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}

	userIDs, userCreated, err := seedUsers(ctx, db, enc, rng, start, string(hash))
	if err != nil {
		return err
	}

	if err := seedOrders(ctx, db, enc, rng, start, now, products, heavy, userIDs, userCreated); err != nil {
		return err
	}
	if err := seedAbandonedCarts(ctx, db, rng, start, products, userIDs); err != nil {
		return err
	}
	if err := seedReviews(ctx, db, rng, start, products, heavy, lowRev, userIDs, userCreated); err != nil {
		return err
	}
	if err := seedActivity(ctx, db, rng, start, products, userIDs, userCreated); err != nil {
		return err
	}
	return nil
}

func loadCatalog(ctx context.Context, db *pgxpool.Pool) ([]histProduct, []string, error) {
	rows, err := db.Query(ctx,
		`SELECT p.id, p.price, p.category_id FROM products p
		 WHERE p.category_id IS NOT NULL ORDER BY p.name`)
	if err != nil {
		return nil, nil, fmt.Errorf("load products: %w", err)
	}
	defer rows.Close()

	var products []histProduct
	seen := map[string]bool{}
	var categories []string
	for rows.Next() {
		var p histProduct
		if err := rows.Scan(&p.id, &p.price, &p.categoryID); err != nil {
			return nil, nil, err
		}
		products = append(products, p)
		if !seen[p.categoryID] {
			seen[p.categoryID] = true
			categories = append(categories, p.categoryID)
		}
	}
	return products, categories, rows.Err()
}

// dayWeighted picks a timestamp in the window with growth (later days more
// likely) and a weekend dip, at a plausible shopping hour.
func dayWeighted(rng *rand.Rand, start time.Time, days int) time.Time {
	for {
		day := int(float64(days) * (1 - rng.Float64()*rng.Float64())) // skew late
		if day >= days {
			day = days - 1
		}
		t := start.AddDate(0, 0, day)
		if wd := t.Weekday(); (wd == time.Saturday || wd == time.Sunday) && rng.Float64() < 0.35 {
			continue // weekend dip
		}
		hour := 8 + rng.Intn(15) // 08:00–22:59
		return time.Date(t.Year(), t.Month(), t.Day(), hour, rng.Intn(60), rng.Intn(60), 0, time.UTC)
	}
}

func seedUsers(ctx context.Context, db *pgxpool.Pool, enc *crypto.Encryptor, rng *rand.Rand, start time.Time, hash string) ([]string, map[string]time.Time, error) {
	ids := make([]string, 0, historyUsers)
	created := make(map[string]time.Time, historyUsers)
	for i := 0; i < historyUsers; i++ {
		id := fmt.Sprintf("f0000000-0000-4000-8000-%012d", i)
		email := strings.ToLower(fmt.Sprintf("seed-user-%d@example.com", i))
		encEmail, err := enc.Encrypt(email)
		if err != nil {
			return nil, nil, fmt.Errorf("encrypt %s: %w", email, err)
		}
		at := dayWeighted(rng, start, historyDays)
		if _, err := db.Exec(ctx,
			`INSERT INTO users (id, email_encrypted, email_hmac, password_hash, role, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, 'customer', $5, $5) ON CONFLICT DO NOTHING`,
			id, encEmail, enc.HMAC(email), hash, at); err != nil {
			return nil, nil, fmt.Errorf("insert user %d: %w", i, err)
		}
		ids = append(ids, id)
		created[id] = at
	}
	return ids, created, nil
}

func pickProduct(rng *rand.Rand, products []histProduct, heavy map[string]bool) histProduct {
	for {
		p := products[rng.Intn(len(products))]
		// Heavy categories win ~3x as often.
		if heavy[p.categoryID] || rng.Float64() < 0.35 {
			return p
		}
	}
}

func orderStatus(rng *rand.Rand) string {
	r := rng.Float64()
	switch {
	case r < 0.40:
		return "delivered"
	case r < 0.58:
		return "shipped"
	case r < 0.72:
		return "paid"
	case r < 0.82:
		return "pending_payment"
	case r < 0.90:
		return "payment_failed"
	case r < 0.96:
		return "cancelled"
	default:
		return "refunded"
	}
}

func seedOrders(ctx context.Context, db *pgxpool.Pool, enc *crypto.Encryptor, rng *rand.Rand,
	start, now time.Time, products []histProduct, heavy map[string]bool,
	userIDs []string, userCreated map[string]time.Time) error {

	for i := 0; i < historyOrders; i++ {
		orderID := fmt.Sprintf("e0000000-0000-4000-8000-%012d", i)
		at := dayWeighted(rng, start, historyDays)

		var userID, guestID *string
		var email string
		maxItems := 2 // guests buy less — the engineered AOV gap
		if rng.Float64() < 0.70 {
			uid := userIDs[rng.Intn(len(userIDs))]
			if uc := userCreated[uid]; at.Before(uc) {
				at = uc.Add(time.Duration(1+rng.Intn(72)) * time.Hour)
			}
			userID = &uid
			email = fmt.Sprintf("seed-user-%s@example.com", uid[len(uid)-4:])
			maxItems = 4
		} else {
			gid := fmt.Sprintf("d0000000-0000-4000-8000-%012d", i)
			guestID = &gid
			email = fmt.Sprintf("guest-%d@example.com", i)
		}
		if at.After(now) {
			at = now.Add(-time.Duration(rng.Intn(3600)) * time.Second)
		}

		encEmail, err := enc.Encrypt(email)
		if err != nil {
			return fmt.Errorf("encrypt order email: %w", err)
		}

		nItems := 1 + rng.Intn(maxItems)
		type line struct {
			p   histProduct
			qty int
		}
		lines := make([]line, 0, nItems)
		subtotal := int64(0)
		for j := 0; j < nItems; j++ {
			p := pickProduct(rng, products, heavy)
			qty := 1 + rng.Intn(3)
			lines = append(lines, line{p, qty})
			subtotal += p.price * int64(qty)
		}
		shipping, method := int64(500), "standard"
		if rng.Float64() < 0.25 {
			shipping, method = 1500, "express"
		}

		tag, err := db.Exec(ctx,
			`INSERT INTO orders (id, order_number, user_id, guest_session_id, status,
			   email_encrypted, subtotal_cents, shipping_cents, total_cents,
			   shipping_method, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
			 ON CONFLICT DO NOTHING`,
			orderID, fmt.Sprintf("HIST-%06d", i), userID, guestID, orderStatus(rng),
			encEmail, subtotal, shipping, subtotal+shipping, method, at)
		if err != nil {
			return fmt.Errorf("insert order %d: %w", i, err)
		}
		if tag.RowsAffected() == 0 {
			continue // already seeded on a previous run
		}

		for j, l := range lines {
			if _, err := db.Exec(ctx,
				`INSERT INTO order_items (id, order_id, product_id, product_name, unit_price_cents, quantity, created_at)
				 SELECT $1, $2, p.id, p.name, $3, $4, $5 FROM products p WHERE p.id = $6
				 ON CONFLICT DO NOTHING`,
				fmt.Sprintf("e1%06d-0000-4000-8000-%012d", i, j), orderID, l.p.price, l.qty, at, l.p.id); err != nil {
				return fmt.Errorf("insert order item %d/%d: %w", i, j, err)
			}
		}
	}
	return nil
}

// seedAbandonedCarts creates stale guest carts with items and no matching
// order — the numerator of the cart-abandonment panel.
func seedAbandonedCarts(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand, start time.Time, products []histProduct, userIDs []string) error {
	for i := 0; i < historyCarts; i++ {
		cartID := fmt.Sprintf("c1000000-0000-4000-8000-%012d", i)
		at := dayWeighted(rng, start, historyDays)

		var userID, guestID *string
		if i < historyCarts/3 {
			// Registered abandoners, from the tail of the user list so they
			// don't collide with the unique one-cart-per-user index.
			uid := userIDs[len(userIDs)-1-i]
			userID = &uid
		} else {
			gid := fmt.Sprintf("d1000000-0000-4000-8000-%012d", i)
			guestID = &gid
		}

		// Draw the items up front so the RNG advances the same amount whether
		// or not the cart already exists — keeps re-runs deterministic.
		type cartLine struct {
			p   histProduct
			qty int
		}
		nItems := 1 + rng.Intn(3)
		items := make([]cartLine, nItems)
		for j := range items {
			items[j] = cartLine{products[rng.Intn(len(products))], 1 + rng.Intn(2)}
		}

		if _, err := db.Exec(ctx,
			`INSERT INTO carts (id, user_id, guest_session_id, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $4) ON CONFLICT DO NOTHING`,
			cartID, userID, guestID, at); err != nil {
			return fmt.Errorf("insert cart %d: %w", i, err)
		}
		for j, l := range items {
			if _, err := db.Exec(ctx,
				`INSERT INTO cart_items (cart_id, product_id, quantity, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $4) ON CONFLICT DO NOTHING`,
				cartID, l.p.id, l.qty, at); err != nil {
				return fmt.Errorf("insert cart item %d/%d: %w", i, j, err)
			}
		}
	}
	return nil
}

func seedReviews(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand, start time.Time,
	products []histProduct, heavy map[string]bool, lowRev string,
	userIDs []string, userCreated map[string]time.Time) error {

	for i := 0; i < historyReviews; i++ {
		uid := userIDs[rng.Intn(len(userIDs))]
		p := products[rng.Intn(len(products))]

		// Engineered divergence: big sellers rate mediocre, the quiet
		// category rates excellent, everything else lands in between.
		var rating int
		switch {
		case p.categoryID == lowRev:
			rating = 4 + rng.Intn(2) // 4–5
		case heavy[p.categoryID]:
			rating = 2 + rng.Intn(3) // 2–4
		default:
			rating = 3 + rng.Intn(3) // 3–5
		}

		at := dayWeighted(rng, start, historyDays)
		if uc := userCreated[uid]; at.Before(uc) {
			at = uc.Add(time.Duration(1+rng.Intn(240)) * time.Hour)
		}

		comment := reviewComments[rng.Intn(len(reviewComments))]
		if _, err := db.Exec(ctx,
			`INSERT INTO reviews (user_id, product_id, rating, comment, status, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, 'approved', $5, $5)
			 ON CONFLICT (user_id, product_id) DO NOTHING`,
			uid, p.id, rating, comment, at); err != nil {
			return fmt.Errorf("insert review %d: %w", i, err)
		}
	}
	return nil
}

// seedActivity backfills user_activity so the conversion-funnel panels have
// history: every seeded order item gets the views and the add_to_cart that
// led to it (plus a purchase event when the order actually got paid), and
// historySessions browse-only sessions provide the top-of-funnel drop-off.
// Engineered shape: roughly 100 views : 30 adds : 15 checkouts : 11 paid.
func seedActivity(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand,
	start time.Time, products []histProduct, userIDs []string, userCreated map[string]time.Time) error {

	// No natural key for ON CONFLICT — a sentinel row marks a completed run.
	// Everything runs in one transaction so an interrupted run leaves no
	// partial rows to duplicate on retry.
	var exists bool
	if err := db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM user_activity WHERE guest_session_id = $1)`,
		activitySentinel).Scan(&exists); err != nil {
		return fmt.Errorf("check activity sentinel: %w", err)
	}
	if exists {
		return nil
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin activity tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	insert := func(userID, guestID *string, event, productID, categoryID string, at time.Time) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO user_activity (user_id, guest_session_id, event_type, product_id, category_id, occurred_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			userID, guestID, event, productID, categoryID, at)
		return err
	}

	// Converting journeys: views + add_to_cart shortly before each seeded
	// order, purchase events for the orders that reached a paid state.
	rows, err := tx.Query(ctx,
		`SELECT o.user_id, o.guest_session_id, o.status, o.created_at, i.product_id, p.category_id
		 FROM orders o
		 JOIN order_items i ON i.order_id = o.id AND i.product_id IS NOT NULL
		 JOIN products p ON p.id = i.product_id
		 WHERE o.order_number LIKE 'HIST-%'`)
	if err != nil {
		return fmt.Errorf("load seeded orders for activity: %w", err)
	}
	defer rows.Close()

	type journey struct {
		userID, guestID *string
		status          string
		at              time.Time
		productID       string
		categoryID      string
	}
	var journeys []journey
	for rows.Next() {
		var j journey
		if err := rows.Scan(&j.userID, &j.guestID, &j.status, &j.at, &j.productID, &j.categoryID); err != nil {
			return err
		}
		journeys = append(journeys, j)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close() // tx uses one connection — release it before the inserts

	paidStates := map[string]bool{"paid": true, "shipped": true, "delivered": true, "refunded": true}
	for _, j := range journeys {
		views := 1 + rng.Intn(3)
		for v := 0; v < views; v++ {
			at := j.at.Add(-time.Duration(2+rng.Intn(40)) * time.Minute)
			if err := insert(j.userID, j.guestID, "view", j.productID, j.categoryID, at); err != nil {
				return fmt.Errorf("insert view activity: %w", err)
			}
		}
		if err := insert(j.userID, j.guestID, "add_to_cart", j.productID, j.categoryID,
			j.at.Add(-time.Duration(1+rng.Intn(15))*time.Minute)); err != nil {
			return fmt.Errorf("insert add_to_cart activity: %w", err)
		}
		if paidStates[j.status] {
			if err := insert(j.userID, j.guestID, "purchase", j.productID, j.categoryID, j.at); err != nil {
				return fmt.Errorf("insert purchase activity: %w", err)
			}
		}
	}

	// Browse-only sessions — the honest top of the funnel that never buys.
	for i := 0; i < historySessions; i++ {
		var userID, guestID *string
		at := dayWeighted(rng, start, historyDays)
		if rng.Float64() < 0.25 {
			uid := userIDs[rng.Intn(len(userIDs))]
			if uc := userCreated[uid]; at.Before(uc) {
				at = uc.Add(time.Duration(1+rng.Intn(48)) * time.Hour)
			}
			userID = &uid
		} else {
			gid := fmt.Sprintf("d2000000-0000-4000-8000-%012d", i)
			guestID = &gid
		}

		views := 1 + rng.Intn(5)
		for v := 0; v < views; v++ {
			p := products[rng.Intn(len(products))]
			if err := insert(userID, guestID, "view", p.id, p.categoryID,
				at.Add(time.Duration(v*(1+rng.Intn(5)))*time.Minute)); err != nil {
				return fmt.Errorf("insert session view: %w", err)
			}
		}
		// A quarter of sessions add to cart and still walk away.
		if rng.Float64() < 0.25 {
			p := products[rng.Intn(len(products))]
			if err := insert(userID, guestID, "add_to_cart", p.id, p.categoryID,
				at.Add(time.Duration(views+2)*time.Minute)); err != nil {
				return fmt.Errorf("insert session add_to_cart: %w", err)
			}
		}
	}

	sentinel := activitySentinel
	if err := insert(nil, &sentinel, "view", products[0].id, products[0].categoryID, start); err != nil {
		return fmt.Errorf("insert activity sentinel: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit activity tx: %w", err)
	}
	return nil
}

var reviewComments = []string{
	"Does what it promises. Would buy again.",
	"Solid quality for the price.",
	"Arrived quickly, works as described.",
	"Decent, though the packaging could be better.",
	"Exceeded my expectations.",
	"Fine for everyday use, nothing spectacular.",
	"Great value — recommended.",
	"Had higher hopes based on the photos.",
	"Exactly as pictured. No complaints.",
	"Good product, slow delivery.",
}
