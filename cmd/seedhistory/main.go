// seedhistory backfills ~60 days of users, orders, abandoned carts, reviews,
// and funnel activity so the Grafana business dashboards have historical data.
//
//	make seed-history            (host, reads .env)
//
// Requires DATABASE_URL and ENCRYPTION_KEY; both come from .env like the API.
// Idempotent — re-running after a `make reset` recreates the same dataset.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/crypto"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/seed"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	key := os.Getenv("ENCRYPTION_KEY")
	if dbURL == "" || key == "" {
		log.Fatal("seedhistory: DATABASE_URL and ENCRYPTION_KEY are required (set them in .env)")
	}

	enc, err := crypto.NewEncryptor(key)
	if err != nil {
		log.Fatalf("seedhistory: init encryptor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("seedhistory: connect: %v", err)
	}
	defer db.Close()

	start := time.Now()
	if err := seed.History(ctx, db, enc); err != nil {
		log.Fatalf("seedhistory: %v", err)
	}

	var users, orders, carts, reviews, activity int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&users)
	_ = db.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&orders)
	_ = db.QueryRow(ctx, `SELECT count(*) FROM carts`).Scan(&carts)
	_ = db.QueryRow(ctx, `SELECT count(*) FROM reviews`).Scan(&reviews)
	_ = db.QueryRow(ctx, `SELECT count(*) FROM user_activity`).Scan(&activity)
	fmt.Printf("seedhistory: done in %s — totals: %d users, %d orders, %d carts, %d reviews, %d activity events\n",
		time.Since(start).Round(time.Millisecond), users, orders, carts, reviews, activity)
}
