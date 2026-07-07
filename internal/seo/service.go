// service.go — assembles the sitemap URL set from static routes + catalog.
package seo

import (
	"context"
	"strings"
	"time"
)

type URL struct {
	Loc     string
	LastMod string // YYYY-MM-DD, empty = omitted
}

type Service interface {
	SitemapURLs(ctx context.Context) ([]URL, error)
}

type service struct {
	repo    Repository
	baseURL string
}

func NewService(repo Repository, baseURL string) Service {
	return &service{repo: repo, baseURL: strings.TrimRight(baseURL, "/")}
}

func (s *service) SitemapURLs(ctx context.Context) ([]URL, error) {
	urls := []URL{
		{Loc: s.baseURL + "/"},
		{Loc: s.baseURL + "/about"},
		{Loc: s.baseURL + "/contact"},
	}

	slugs, err := s.repo.CategorySlugs(ctx)
	if err != nil {
		return nil, err
	}
	for _, slug := range slugs {
		urls = append(urls, URL{Loc: s.baseURL + "/shop/" + slug})
	}

	products, err := s.repo.ProductEntries(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range products {
		urls = append(urls, URL{
			Loc:     s.baseURL + "/product/" + p.ID,
			LastMod: p.UpdatedAt.UTC().Format(time.DateOnly),
		})
	}
	return urls, nil
}
