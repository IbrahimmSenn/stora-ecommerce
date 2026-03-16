package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// --- Mock repository ---

type mockRepo struct {
	users map[string]User
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: make(map[string]User)}
}

func (m *mockRepo) CreateUser(_ context.Context, u User) error {
	if _, ok := m.users[u.Email]; ok {
		return ErrEmailExists
	}
	m.users[u.Email] = u
	return nil
}

func (m *mockRepo) GetUserByEmail(_ context.Context, email string) (*User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

func (m *mockRepo) GetUserByID(_ context.Context, id string) (*User, error) {
	for _, u := range m.users {
		if u.Id.String() == id {
			return &u, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *mockRepo) CreateOAuthUser(_ context.Context, u User) error {
	if _, ok := m.users[u.Email]; ok {
		return ErrEmailExists
	}
	m.users[u.Email] = u
	return nil
}

func (m *mockRepo) UpdatePassword(_ context.Context, userID string, passwordHash string) error {
	for email, u := range m.users {
		if u.Id.String() == userID {
			u.PasswordHash = passwordHash
			m.users[email] = u
			return nil
		}
	}
	return ErrUserNotFound
}

// --- Input validation tests ---

func TestRegister_ValidInput(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil)

	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "user@example.com",
		Password: "securepassword",
	})
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", resp.Email)
	assert.NotEmpty(t, resp.Id)
}

func TestRegister_EmptyEmail(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "",
		Password: "securepassword",
	})
	assert.Error(t, err)
}

func TestRegister_InvalidEmailFormat(t *testing.T) {
	tests := []string{
		"notanemail",
		"@missing-local.com",
		"missing-domain@",
		"spaces in@email.com",
		"missing@.com",
	}

	for _, email := range tests {
		t.Run(email, func(t *testing.T) {
			svc := NewService(newMockRepo(), bcrypt.MinCost, nil)
			_, err := svc.Register(context.Background(), RegisterRequest{
				Email:    email,
				Password: "securepassword",
			})
			assert.Error(t, err, "should reject invalid email: %s", email)
		})
	}
}

func TestRegister_EmptyPassword(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "user@example.com",
		Password: "",
	})
	assert.Error(t, err)
}

func TestRegister_ShortPassword(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "user@example.com",
		Password: "short",
	})
	assert.Error(t, err, "passwords under 8 characters should be rejected")
}

func TestRegister_ExactMinPassword(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "user@example.com",
		Password: "12345678",
	})
	assert.NoError(t, err, "exactly 8 characters should be accepted")
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, bcrypt.MinCost, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email: "dup@example.com", Password: "securepassword",
	})
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), RegisterRequest{
		Email: "dup@example.com", Password: "securepassword",
	})
	assert.ErrorIs(t, err, ErrEmailExists)
}

func TestRegister_EmailNormalization_Case(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil)

	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "USER@EXAMPLE.COM",
		Password: "securepassword",
	})
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", resp.Email, "email should be lowercased")
}

func TestRegister_EmailWithSpaces_Rejected(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "  user@example.com  ",
		Password: "securepassword",
	})
	assert.Error(t, err, "email with leading/trailing spaces should be rejected by validator")
}

func TestRegister_BothFieldsEmpty(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "",
		Password: "",
	})
	assert.Error(t, err)
}
