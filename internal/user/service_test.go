package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
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

func (m *mockRepo) UpdateName(_ context.Context, userID string, name string) error {
	for email, u := range m.users {
		if u.Id.String() == userID {
			u.Name = name
			m.users[email] = u
			return nil
		}
	}
	return ErrUserNotFound
}

func (m *mockRepo) ListAll(_ context.Context, limit, offset int) ([]User, int, error) {
	out := []User{}
	for _, u := range m.users {
		out = append(out, u)
	}
	return out, len(out), nil
}

func (m *mockRepo) UpdateRole(_ context.Context, userID, role string) error {
	for email, u := range m.users {
		if u.Id.String() == userID {
			u.Role = role
			m.users[email] = u
			return nil
		}
	}
	return ErrUserNotFound
}

func (m *mockRepo) CountByRole(_ context.Context, role string) (int, error) {
	n := 0
	for _, u := range m.users {
		if u.Role == role {
			n++
		}
	}
	return n, nil
}

// --- Input validation tests ---

func TestRegister_ValidInput(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil, nil)

	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "user@example.com",
		Password: "Securepass1!",
	})
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", resp.Email)
	assert.NotEmpty(t, resp.Id)
}

func TestRegister_EmptyEmail(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "",
		Password: "Securepass1!",
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
			svc := NewService(newMockRepo(), bcrypt.MinCost, nil, nil)
			_, err := svc.Register(context.Background(), RegisterRequest{
				Email:    email,
				Password: "Securepass1!",
			})
			assert.Error(t, err, "should reject invalid email: %s", email)
		})
	}
}

func TestRegister_EmptyPassword(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "user@example.com",
		Password: "",
	})
	assert.Error(t, err)
}

func TestRegister_ShortPassword(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "user@example.com",
		Password: "short",
	})
	assert.Error(t, err, "passwords under 8 characters should be rejected")
}

func TestRegister_ExactMinPassword(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "user@example.com",
		Password: "Ab1!cdef",
	})
	assert.NoError(t, err, "exactly 8 characters should be accepted")
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, bcrypt.MinCost, nil, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email: "dup@example.com", Password: "Securepass1!",
	})
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), RegisterRequest{
		Email: "dup@example.com", Password: "Securepass1!",
	})
	assert.ErrorIs(t, err, ErrEmailExists)
}

func TestRegister_EmailNormalization_Case(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil, nil)

	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "USER@EXAMPLE.COM",
		Password: "Securepass1!",
	})
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", resp.Email, "email should be lowercased")
}

func TestRegister_EmailWithSpaces_Rejected(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "  user@example.com  ",
		Password: "Securepass1!",
	})
	assert.Error(t, err, "email with leading/trailing spaces should be rejected by validator")
}

func TestRegister_BothFieldsEmpty(t *testing.T) {
	svc := NewService(newMockRepo(), bcrypt.MinCost, nil, nil)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "",
		Password: "",
	})
	assert.Error(t, err)
}

// --- Profile tests ---

type mockRevoker struct {
	revoked []string
}

func (m *mockRevoker) RevokeAllUserTokens(_ context.Context, userID string) error {
	m.revoked = append(m.revoked, userID)
	return nil
}

func registerTestUser(t *testing.T, svc UserService) *UserResponse {
	t.Helper()
	resp, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "profile@example.com",
		Password: "Securepass1!",
	})
	require.NoError(t, err)
	return resp
}

func TestGetMe_ReturnsProfile(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, bcrypt.MinCost, nil, nil)
	u := registerTestUser(t, svc)

	me, err := svc.GetMe(context.Background(), u.Id.String())
	require.NoError(t, err)
	assert.Equal(t, u.Email, me.Email)
	assert.Empty(t, me.Name)
}

func TestUpdateProfile_TrimsAndStores(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, bcrypt.MinCost, nil, nil)
	u := registerTestUser(t, svc)

	me, err := svc.UpdateProfile(context.Background(), u.Id.String(), UpdateProfileRequest{Name: "  Ada Lovelace  "})
	require.NoError(t, err)
	assert.Equal(t, "Ada Lovelace", me.Name)
}

func TestUpdateProfile_EmptyClears(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, bcrypt.MinCost, nil, nil)
	u := registerTestUser(t, svc)

	_, err := svc.UpdateProfile(context.Background(), u.Id.String(), UpdateProfileRequest{Name: "Ada"})
	require.NoError(t, err)
	me, err := svc.UpdateProfile(context.Background(), u.Id.String(), UpdateProfileRequest{Name: ""})
	require.NoError(t, err)
	assert.Empty(t, me.Name)
}

func TestUpdateProfile_TooLong(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, bcrypt.MinCost, nil, nil)
	u := registerTestUser(t, svc)

	long := make([]rune, 101)
	for i := range long {
		long[i] = 'a'
	}
	_, err := svc.UpdateProfile(context.Background(), u.Id.String(), UpdateProfileRequest{Name: string(long)})
	assert.ErrorIs(t, err, ErrNameTooLong)
}

func TestChangePassword_WrongCurrent(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, bcrypt.MinCost, nil, nil)
	u := registerTestUser(t, svc)

	err := svc.ChangePassword(context.Background(), u.Id.String(), ChangePasswordRequest{
		CurrentPassword: "not-the-password",
		NewPassword:     "Newsecurepass1!",
	})
	assert.ErrorIs(t, err, ErrWrongPassword)
}

func TestChangePassword_WeakNew(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, bcrypt.MinCost, nil, nil)
	u := registerTestUser(t, svc)

	err := svc.ChangePassword(context.Background(), u.Id.String(), ChangePasswordRequest{
		CurrentPassword: "Securepass1!",
		NewPassword:     "weak",
	})
	assert.Error(t, err)
}

func TestChangePassword_OAuthAccountRejected(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, bcrypt.MinCost, nil, nil)
	oauthUser := User{Id: uuid.New(), Email: "oauth@example.com", PasswordHash: ""}
	require.NoError(t, repo.CreateOAuthUser(context.Background(), oauthUser))

	err := svc.ChangePassword(context.Background(), oauthUser.Id.String(), ChangePasswordRequest{
		CurrentPassword: "anything",
		NewPassword:     "Newsecurepass1!",
	})
	assert.ErrorIs(t, err, ErrNoPassword)
}

func TestChangePassword_SuccessUpdatesHashAndRevokes(t *testing.T) {
	repo := newMockRepo()
	revoker := &mockRevoker{}
	svc := NewService(repo, bcrypt.MinCost, nil, revoker)
	u := registerTestUser(t, svc)

	err := svc.ChangePassword(context.Background(), u.Id.String(), ChangePasswordRequest{
		CurrentPassword: "Securepass1!",
		NewPassword:     "Newsecurepass1!",
	})
	require.NoError(t, err)

	stored, err := repo.GetUserByID(context.Background(), u.Id.String())
	require.NoError(t, err)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("Newsecurepass1!")))
	assert.Equal(t, []string{u.Id.String()}, revoker.revoked)
}
