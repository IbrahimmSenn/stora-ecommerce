package address

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	lastReq AddressRequest
	called  bool
}

func (m *mockRepo) Create(_ context.Context, _ string, req AddressRequest) (*Address, error) {
	m.called = true
	m.lastReq = req
	return &Address{RecipientName: req.RecipientName, Country: req.Country}, nil
}
func (m *mockRepo) List(_ context.Context, _ string) ([]Address, error) { return nil, nil }
func (m *mockRepo) Update(_ context.Context, _, _ string, _ AddressRequest) (*Address, error) {
	return nil, nil
}
func (m *mockRepo) Delete(_ context.Context, _, _ string) error     { return nil }
func (m *mockRepo) SetDefault(_ context.Context, _, _ string) error { return nil }

func validReq() AddressRequest {
	return AddressRequest{
		RecipientName: "Jane Buyer",
		Line1:         "12 Oak St",
		City:          "Tallinn",
		Region:        "Harju",
		PostalCode:    "10115",
		Country:       "ee", // lower-case on purpose
	}
}

func TestCreate_NormalisesAndDelegates(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), "user-1", validReq())
	require.NoError(t, err)
	assert.True(t, repo.called)
	assert.Equal(t, "EE", repo.lastReq.Country, "country should be upper-cased for a stable lookup")
}

func TestCreate_RejectsInvalidCountry(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	req := validReq()
	req.Country = "USA" // not 2-letter
	_, err := svc.Create(context.Background(), "user-1", req)

	assert.Error(t, err)
	assert.False(t, repo.called, "validation must fail before hitting the repo")
}

func TestCreate_RejectsMissingFields(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo)

	req := validReq()
	req.RecipientName = ""
	_, err := svc.Create(context.Background(), "user-1", req)

	assert.Error(t, err)
	assert.False(t, repo.called)
}
