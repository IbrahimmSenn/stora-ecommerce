package category

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBuildTree_Flat(t *testing.T) {
	cats := []Category{
		{ID: uuid.New(), Name: "Electronics", Slug: "electronics"},
		{ID: uuid.New(), Name: "Clothing", Slug: "clothing"},
	}

	tree := buildTree(cats)
	assert.Len(t, tree, 2)
	assert.Empty(t, tree[0].Children)
	assert.Empty(t, tree[1].Children)
}

func TestBuildTree_Nested(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()
	grandchildID := uuid.New()

	cats := []Category{
		{ID: parentID, Name: "Electronics", Slug: "electronics"},
		{ID: childID, Name: "Laptops", Slug: "laptops", ParentID: &parentID},
		{ID: grandchildID, Name: "Gaming Laptops", Slug: "gaming-laptops", ParentID: &childID},
	}

	tree := buildTree(cats)
	assert.Len(t, tree, 1, "should have one root")
	assert.Equal(t, "Electronics", tree[0].Name)
	assert.Len(t, tree[0].Children, 1)
	assert.Equal(t, "Laptops", tree[0].Children[0].Name)
	assert.Len(t, tree[0].Children[0].Children, 1)
	assert.Equal(t, "Gaming Laptops", tree[0].Children[0].Children[0].Name)
}

func TestBuildTree_Empty(t *testing.T) {
	tree := buildTree(nil)
	assert.Nil(t, tree)
}

func TestBuildTree_OrphanedChild(t *testing.T) {
	// A child whose parent_id doesn't match any category becomes a root.
	missingParent := uuid.New()
	cats := []Category{
		{ID: uuid.New(), Name: "Orphan", Slug: "orphan", ParentID: &missingParent},
	}

	tree := buildTree(cats)
	assert.Len(t, tree, 1, "orphan should become a root node")
}
