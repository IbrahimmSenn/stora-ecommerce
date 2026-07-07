package seo

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepo struct {
	products []ProductEntry
	slugs    []string
	err      error
}

func (f *fakeRepo) ProductEntries(ctx context.Context) ([]ProductEntry, error) {
	return f.products, f.err
}

func (f *fakeRepo) CategorySlugs(ctx context.Context) ([]string, error) {
	return f.slugs, f.err
}

func TestSitemapURLs(t *testing.T) {
	updated := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{
		products: []ProductEntry{{ID: "p1", UpdatedAt: updated}},
		slugs:    []string{"electronics"},
	}
	// trailing slash on baseURL must not produce double slashes
	svc := NewService(repo, "https://shop.example.com/")

	urls, err := svc.SitemapURLs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"https://shop.example.com/":                 "",
		"https://shop.example.com/about":            "",
		"https://shop.example.com/contact":          "",
		"https://shop.example.com/shop/electronics": "",
		"https://shop.example.com/product/p1":       "2026-07-01",
	}
	if len(urls) != len(want) {
		t.Fatalf("got %d urls, want %d: %+v", len(urls), len(want), urls)
	}
	for _, u := range urls {
		lastmod, ok := want[u.Loc]
		if !ok {
			t.Errorf("unexpected url %q", u.Loc)
			continue
		}
		if u.LastMod != lastmod {
			t.Errorf("url %q lastmod = %q, want %q", u.Loc, u.LastMod, lastmod)
		}
	}
}

func TestSitemapURLs_RepoError(t *testing.T) {
	svc := NewService(&fakeRepo{err: errors.New("db down")}, "https://shop.example.com")
	if _, err := svc.SitemapURLs(context.Background()); err == nil {
		t.Fatal("expected error when repository fails")
	}
}
