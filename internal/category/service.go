// service.go — category logic. Converts the flat DB rows into a nested tree for the API.
package category

import (
	"context"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/cache"
)

type Service interface {
	List(ctx context.Context) ([]Category, error)
	ListTree(ctx context.Context) ([]CategoryTree, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	GetBySlug(ctx context.Context, slug string) (*Category, error)
	Create(ctx context.Context, req CreateCategoryRequest) (*Category, error)
	Update(ctx context.Context, id string, req UpdateCategoryRequest) (*Category, error)
	Delete(ctx context.Context, id string) error
}

const (
	cacheKeyList = "category:list"
	cacheKeyTree = "category:tree"
)

type service struct {
	repo     Repository
	validate *validator.Validate
	// cache is optional (nil = no caching). The category tree changes only on
	// admin edits, so a short TTL is safe and keeps it off the hot read path.
	cache    cache.Cache
	cacheTTL time.Duration
}

func NewService(repo Repository) Service {
	return &service{repo: repo, validate: validator.New()}
}

// NewServiceWithCache adds a read cache for the category list/tree. Pass the
// app's shared cache (in-memory by default, Redis when configured).
func NewServiceWithCache(repo Repository, c cache.Cache, ttl time.Duration) Service {
	return &service{repo: repo, validate: validator.New(), cache: c, cacheTTL: ttl}
}

func (s *service) List(ctx context.Context) ([]Category, error) {
	if s.cache != nil {
		if v, ok := cache.GetJSON[[]Category](ctx, s.cache, cacheKeyList); ok {
			return v, nil
		}
	}
	cats, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = cache.SetJSON(ctx, s.cache, cacheKeyList, cats, s.cacheTTL)
	}
	return cats, nil
}

// ListTree returns categories organized as a nested tree structure
// for intuitive browsing (top-level categories with their children).
func (s *service) ListTree(ctx context.Context) ([]CategoryTree, error) {
	if s.cache != nil {
		if v, ok := cache.GetJSON[[]CategoryTree](ctx, s.cache, cacheKeyTree); ok {
			return v, nil
		}
	}
	cats, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	tree := buildTree(cats)
	if s.cache != nil {
		_ = cache.SetJSON(ctx, s.cache, cacheKeyTree, tree, s.cacheTTL)
	}
	return tree, nil
}

func (s *service) GetByID(ctx context.Context, id string) (*Category, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetBySlug(ctx context.Context, slug string) (*Category, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *service) Create(ctx context.Context, req CreateCategoryRequest) (*Category, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	var parentID *uuid.UUID
	if req.ParentID != nil {
		parsed, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent_id: %w", err)
		}
		parentID = &parsed
	}

	c, err := s.repo.Create(ctx, req.Name, req.Slug, parentID)
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	s.invalidate(ctx)
	return c, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateCategoryRequest) (*Category, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	var parentID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		parsed, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent_id: %w", err)
		}
		// A category cannot be its own parent.
		if parsed.String() == id {
			return nil, fmt.Errorf("category cannot be its own parent")
		}
		parentID = &parsed
	}

	c, err := s.repo.Update(ctx, id, req.Name, req.Slug, parentID)
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx)
	return c, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidate(ctx)
	return nil
}

// invalidate drops the cached list/tree so a new category shows immediately
// rather than waiting out the TTL.
func (s *service) invalidate(ctx context.Context) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Delete(ctx, cacheKeyList)
	_ = s.cache.Delete(ctx, cacheKeyTree)
}

// buildTree converts a flat list of categories into a nested tree.
func buildTree(cats []Category) []CategoryTree {
	type node struct {
		CategoryTree
		childPtrs []*node
	}

	nodeMap := make(map[uuid.UUID]*node)
	var roots []*node

	// First pass: create all nodes.
	for _, c := range cats {
		nodeMap[c.ID] = &node{
			CategoryTree: CategoryTree{
				ID:       c.ID,
				Name:     c.Name,
				Slug:     c.Slug,
				ImageURL: c.ImageURL,
			},
		}
	}

	// Second pass: link children to parents via pointers.
	for _, c := range cats {
		n := nodeMap[c.ID]
		if c.ParentID != nil {
			if parent, ok := nodeMap[*c.ParentID]; ok {
				parent.childPtrs = append(parent.childPtrs, n)
				continue
			}
		}
		roots = append(roots, n)
	}

	// Third pass: recursively convert pointer tree to value tree.
	var convert func(n *node) CategoryTree
	convert = func(n *node) CategoryTree {
		ct := n.CategoryTree
		for _, child := range n.childPtrs {
			ct.Children = append(ct.Children, convert(child))
		}
		return ct
	}

	var result []CategoryTree
	for _, r := range roots {
		result = append(result, convert(r))
	}
	return result
}
