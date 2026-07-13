// Command imagevariants backfills size variants for product images that only
// have a legacy `url` (e.g. the seeded catalogue). It reads each source image
// from a local directory, generates thumbnail/card/full variants into
// UPLOAD_DIR, and updates the row. Safe to re-run — rows that already have a
// card_url are skipped.
//
// Usage:
//
//	DATABASE_URL=... UPLOAD_DIR=./uploads go run ./cmd/imagevariants [sourceDir]
//
// sourceDir defaults to web/public/products. Each row's url is expected to look
// like /products/<file>; <file> is resolved against sourceDir.
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/imageproc"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	sourceDir := "web/public/products"
	if len(os.Args) > 1 {
		sourceDir = os.Args[1]
	}

	proc, err := imageproc.New(uploadDir, "/media")
	if err != nil {
		log.Fatalf("image processor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(ctx,
		`SELECT id, url FROM product_images WHERE card_url IS NULL AND url LIKE '/products/%'`)
	if err != nil {
		log.Fatalf("query images: %v", err)
	}

	type job struct{ id, url string }
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.url); err != nil {
			log.Fatalf("scan: %v", err)
		}
		jobs = append(jobs, j)
	}
	rows.Close()

	done, skipped := 0, 0
	for _, j := range jobs {
		// filepath.Base strips any directory components, so src stays inside
		// sourceDir. Offline admin CLI, paths come from our own database.
		base := filepath.Base(strings.TrimPrefix(j.url, "/products/"))
		src := filepath.Join(sourceDir, base)
		f, err := os.Open(src) // #nosec G703
		if err != nil {
			log.Printf("skip %s: %v", j.url, err)
			skipped++
			continue
		}
		v, err := proc.Process(strings.TrimSuffix(base, filepath.Ext(base)), f)
		_ = f.Close()
		if err != nil {
			log.Printf("skip %s: %v", j.url, err)
			skipped++
			continue
		}
		if _, err := db.Exec(ctx,
			`UPDATE product_images SET thumbnail_url=$1, card_url=$2, full_url=$3 WHERE id=$4`,
			v.ThumbnailURL, v.CardURL, v.FullURL, j.id); err != nil {
			log.Printf("update %s: %v", j.id, err)
			skipped++
			continue
		}
		done++
	}

	log.Printf("image variants: %d generated, %d skipped (of %d)", done, skipped, len(jobs))
}
