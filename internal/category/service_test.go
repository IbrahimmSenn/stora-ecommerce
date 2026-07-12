package category

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepo is a hand-rolled Repository stub for service-level tests.
type mockRepo struct {
	updated    *Category
	updateErr  error
	deleteErr  error
	deleteID   string
	deleteCall bool
}

func (m *mockRepo) List(context.Context) ([]Category, error)               { return nil, nil }
func (m *mockRepo) GetByID(context.Context, string) (*Category, error)     { return nil, nil }
func (m *mockRepo) GetBySlug(context.Context, string) (*Category, error)   { return nil, nil }
func (m *mockRepo) Create(context.Context, string, string, *uuid.UUID) (*Category, error) {
	return nil, nil
}
func (m *mockRepo) Update(_ context.Context, id, name, slug string, _ *uuid.UUID) (*Category, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	m.updated = &Category{Name: name, Slug: slug}
	return m.updated, nil
}
func (m *mockRepo) Delete(_ context.Context, id string) error {
	m.deleteCall = true
	m.deleteID = id
	return m.deleteErr
}

func TestService_Update_Valid(t *testing.T) {
	repo := &mockRepo{}
	s := NewService(repo)
	got, err := s.Update(context.Background(), uuid.NewString(), UpdateCategoryRequest{Name: "Books", Slug: "books"})
	assert.NoError(t, err)
	assert.Equal(t, "Books", got.Name)
}

func TestService_Update_RejectsEmptyName(t *testing.T) {
	s := NewService(&mockRepo{})
	_, err := s.Update(context.Background(), uuid.NewString(), UpdateCategoryRequest{Name: "", Slug: "books"})
	assert.Error(t, err, "empty name should fail validation")
}

func TestService_Update_RejectsSelfParent(t *testing.T) {
	id := uuid.NewString()
	s := NewService(&mockRepo{})
	_, err := s.Update(context.Background(), id, UpdateCategoryRequest{Name: "X", Slug: "x", ParentID: &id})
	assert.Error(t, err, "a category cannot be its own parent")
}

func TestService_Delete_PassesThroughInUse(t *testing.T) {
	repo := &mockRepo{deleteErr: ErrCategoryInUse}
	s := NewService(repo)
	err := s.Delete(context.Background(), "abc")
	assert.ErrorIs(t, err, ErrCategoryInUse)
	assert.True(t, repo.deleteCall)
}

func TestBuildTree_Flat(t *testing.T) {
	img := "/products/foo-1.webp"
	cats := []Category{
		{ID: uuid.New(), Name: "Electronics", Slug: "electronics", ImageURL: &img},
		{ID: uuid.New(), Name: "Clothing", Slug: "clothing"},
	}

	tree := buildTree(cats)
	assert.Len(t, tree, 2)
	assert.Empty(t, tree[0].Children)
	assert.Empty(t, tree[1].Children)
	// image_url carries through to the tree; missing stays nil.
	require.NotNil(t, tree[0].ImageURL)
	assert.Equal(t, img, *tree[0].ImageURL)
	assert.Nil(t, tree[1].ImageURL)
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
