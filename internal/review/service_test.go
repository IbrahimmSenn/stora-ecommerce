package review

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validProduct() string { return uuid.NewString() }
func validUser() string    { return uuid.NewString() }

func TestCreate_RejectsWhenNotPurchased(t *testing.T) {
	svc := NewService(&stubReviewRepo{purchased: false})

	_, err := svc.Create(context.Background(), validUser(), validProduct(), CreateReviewRequest{Rating: 5})
	assert.ErrorIs(t, err, ErrNotPurchased)
}

func TestCreate_AllowsWhenPurchased(t *testing.T) {
	svc := NewService(&stubReviewRepo{purchased: true})

	rv, err := svc.Create(context.Background(), validUser(), validProduct(), CreateReviewRequest{Rating: 4})
	require.NoError(t, err)
	assert.Equal(t, 4, rv.Rating)
	assert.Equal(t, StatusApproved, rv.Status, "new reviews are published immediately")
}

func TestCreate_RejectsRatingOutOfRange(t *testing.T) {
	svc := NewService(&stubReviewRepo{purchased: true})

	for _, bad := range []int{0, 6, -1} {
		_, err := svc.Create(context.Background(), validUser(), validProduct(), CreateReviewRequest{Rating: bad})
		assert.Error(t, err, "rating %d should be rejected", bad)
	}
}

func TestCreate_RejectsBadProductID(t *testing.T) {
	svc := NewService(&stubReviewRepo{purchased: true})

	_, err := svc.Create(context.Background(), validUser(), "not-a-uuid", CreateReviewRequest{Rating: 5})
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestList_DefaultsSortAndPaging(t *testing.T) {
	repo := &stubReviewRepo{}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), ListParams{ProductID: validProduct(), Sort: "bogus"})
	require.NoError(t, err)
	assert.Equal(t, SortHelpful, repo.lastList.Sort, "unknown sort should fall back to helpful")
	assert.Equal(t, 1, repo.lastList.Page)
	assert.Equal(t, 10, repo.lastList.PageSize)
}

func TestEligibility_ReportsAlreadyReviewed(t *testing.T) {
	existing := 3
	svc := NewService(&stubReviewRepo{purchased: true, existing: &Review{Rating: existing, Status: StatusPending}})

	e, err := svc.Eligibility(context.Background(), validUser(), validProduct())
	require.NoError(t, err)
	assert.True(t, e.HasPurchased)
	assert.True(t, e.AlreadyReviewed)
	assert.False(t, e.CanReview, "cannot review again once reviewed")
	assert.True(t, e.ExistingPending)
}

func TestSetStatus_RejectsInvalidStatus(t *testing.T) {
	svc := NewService(&stubReviewRepo{})
	err := svc.SetStatus(context.Background(), uuid.NewString(), "garbage")
	assert.Error(t, err)
}

// --- stub ---

type stubReviewRepo struct {
	purchased bool
	existing  *Review
	lastList  ListParams
}

func (s *stubReviewRepo) Create(_ context.Context, userID, productID uuid.UUID, rating int, comment *string) (*Review, error) {
	return &Review{ID: uuid.New(), UserID: userID, ProductID: productID, Rating: rating, Comment: comment, Status: StatusApproved}, nil
}

func (s *stubReviewRepo) ListByProduct(_ context.Context, p ListParams) (*ListResult, error) {
	s.lastList = p
	return &ListResult{Reviews: []PublicReview{}, Distribution: map[int]int{}}, nil
}

func (s *stubReviewRepo) HasPurchased(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return s.purchased, nil
}

func (s *stubReviewRepo) GetUserReview(_ context.Context, _, _ uuid.UUID) (*Review, error) {
	if s.existing != nil {
		return s.existing, nil
	}
	return nil, ErrReviewNotFound
}

func (s *stubReviewRepo) AddVote(_ context.Context, _, _ uuid.UUID) error    { return nil }
func (s *stubReviewRepo) RemoveVote(_ context.Context, _, _ uuid.UUID) error { return nil }

func (s *stubReviewRepo) ListForModeration(_ context.Context, _ string, _, _ int) ([]ModerationItem, int, error) {
	return []ModerationItem{}, 0, nil
}
func (s *stubReviewRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (s *stubReviewRepo) Delete(_ context.Context, _ uuid.UUID) error                 { return nil }
