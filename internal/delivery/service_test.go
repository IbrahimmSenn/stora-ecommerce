package delivery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubRepo struct {
	created   *DeliveryOption
	createErr error
	rateCents int64
	rateOK    bool
}

func (s *stubRepo) List(context.Context, bool) ([]DeliveryOption, error) { return nil, nil }
func (s *stubRepo) Create(_ context.Context, o DeliveryOption) (*DeliveryOption, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	s.created = &o
	return &o, nil
}
func (s *stubRepo) Update(_ context.Context, _ string, o DeliveryOption) (*DeliveryOption, error) {
	return &o, nil
}
func (s *stubRepo) Delete(context.Context, string) error { return nil }
func (s *stubRepo) RateByCode(context.Context, string) (int64, bool, error) {
	return s.rateCents, s.rateOK, nil
}

func boolPtr(b bool) *bool { return &b }

func TestCreate_NormalizesCodeAndDefaultsActive(t *testing.T) {
	repo := &stubRepo{}
	s := NewService(repo)
	got, err := s.Create(context.Background(), CreateRequest{Code: "  NEXT-Day ", Label: "Next day", PriceCents: 2000})
	assert.NoError(t, err)
	assert.Equal(t, "next-day", got.Code, "code lowercased and trimmed")
	assert.True(t, got.Active, "active defaults to true")
}

func TestCreate_RejectsBadCode(t *testing.T) {
	s := NewService(&stubRepo{})
	_, err := s.Create(context.Background(), CreateRequest{Code: "next day!", Label: "X", PriceCents: 1})
	assert.ErrorIs(t, err, ErrInvalidCode)
}

func TestCreate_RejectsNegativePrice(t *testing.T) {
	s := NewService(&stubRepo{})
	_, err := s.Create(context.Background(), CreateRequest{Code: "x", Label: "X", PriceCents: -5})
	assert.Error(t, err, "negative price should fail validation")
}

func TestCreate_HonorsExplicitInactive(t *testing.T) {
	s := NewService(&stubRepo{})
	got, err := s.Create(context.Background(), CreateRequest{Code: "x", Label: "X", PriceCents: 0, Active: boolPtr(false)})
	assert.NoError(t, err)
	assert.False(t, got.Active)
}

func TestRate_DelegatesToRepo(t *testing.T) {
	s := NewService(&stubRepo{rateCents: 1500, rateOK: true})
	cents, ok, err := s.Rate(context.Background(), "express")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(1500), cents)
}
