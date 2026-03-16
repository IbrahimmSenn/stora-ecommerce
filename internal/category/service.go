package category

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Service interface {
	List(ctx context.Context) ([]Category, error)
	ListTree(ctx context.Context) ([]CategoryTree, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	GetBySlug(ctx context.Context, slug string) (*Category, error)
	Create(ctx context.Context, req CreateCategoryRequest) (*Category, error)
}

type service struct {
	repo     Repository
	validate *validator.Validate
}

func NewService(repo Repository) Service {
	return &service{repo: repo, validate: validator.New()}
}

func (s *service) List(ctx context.Context) ([]Category, error) {
	return s.repo.List(ctx)
}

// ListTree returns categories organized as a nested tree structure
// for intuitive browsing (top-level categories with their children).
func (s *service) ListTree(ctx context.Context) ([]CategoryTree, error) {
	cats, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return buildTree(cats), nil
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
	return c, nil
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
				ID:   c.ID,
				Name: c.Name,
				Slug: c.Slug,
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
