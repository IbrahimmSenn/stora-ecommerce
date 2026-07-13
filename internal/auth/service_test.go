package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/user"
)

// --- Mock user repository ---

type mockUserRepo struct {
	users map[string]*user.User // keyed by email
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*user.User)}
}

func (m *mockUserRepo) CreateUser(_ context.Context, u user.User) error {
	if _, ok := m.users[u.Email]; ok {
		return user.ErrEmailExists
	}
	m.users[u.Email] = &u
	return nil
}

func (m *mockUserRepo) GetUserByEmail(_ context.Context, email string) (*user.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetUserByID(_ context.Context, id string) (*user.User, error) {
	for _, u := range m.users {
		if u.Id.String() == id {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepo) CreateOAuthUser(_ context.Context, u user.User) error {
	if _, ok := m.users[u.Email]; ok {
		return user.ErrEmailExists
	}
	m.users[u.Email] = &u
	return nil
}

func (m *mockUserRepo) UpdatePassword(_ context.Context, userID string, passwordHash string) error {
	for _, u := range m.users {
		if u.Id.String() == userID {
			u.PasswordHash = passwordHash
			return nil
		}
	}
	return user.ErrUserNotFound
}

func (m *mockUserRepo) UpdateName(_ context.Context, userID string, name string) error {
	for _, u := range m.users {
		if u.Id.String() == userID {
			u.Name = name
			return nil
		}
	}
	return user.ErrUserNotFound
}

func (m *mockUserRepo) ListAll(_ context.Context, _, _ int) ([]user.User, int, error) {
	return nil, 0, nil
}

func (m *mockUserRepo) UpdateRole(_ context.Context, _, _ string) error { return nil }

func (m *mockUserRepo) CountByRole(_ context.Context, _ string) (int, error) { return 0, nil }

// --- Mock auth repository ---

type mockAuthRepo struct {
	tokens          map[string]*RefreshToken // keyed by token string
	revokedAllCalls []string                 // userIDs passed to RevokeAllUserTokens
}

func newMockAuthRepo() *mockAuthRepo {
	return &mockAuthRepo{tokens: make(map[string]*RefreshToken)}
}

func (m *mockAuthRepo) StoreRefreshToken(_ context.Context, token RefreshToken) error {
	m.tokens[token.Token] = &token
	return nil
}

func (m *mockAuthRepo) GetRefreshToken(_ context.Context, tokenString string) (*RefreshToken, error) {
	t, ok := m.tokens[tokenString]
	if !ok {
		return nil, ErrTokenNotFound
	}
	return t, nil
}

func (m *mockAuthRepo) MarkRefreshTokenUsed(_ context.Context, tokenID string) error {
	for _, t := range m.tokens {
		if t.ID.String() == tokenID {
			// Mirror the atomic "AND used = false" guard: a second consume of
			// the same token reports ErrTokenUsed.
			if t.Used {
				return ErrTokenUsed
			}
			t.Used = true
			return nil
		}
	}
	return ErrTokenNotFound
}

func (m *mockAuthRepo) RevokeAllUserTokens(_ context.Context, userID string) error {
	m.revokedAllCalls = append(m.revokedAllCalls, userID)
	for _, t := range m.tokens {
		if t.UserID.String() == userID {
			t.Revoked = true
		}
	}
	return nil
}

func (m *mockAuthRepo) StorePasswordResetToken(_ context.Context, _ PasswordResetToken) error {
	return nil
}

func (m *mockAuthRepo) GetPasswordResetToken(_ context.Context, _ string) (*PasswordResetToken, error) {
	return nil, ErrResetTokenNotFound
}

func (m *mockAuthRepo) MarkResetTokenUsed(_ context.Context, _ string) error {
	return nil
}

func (m *mockAuthRepo) Store2FA(_ context.Context, _ TwoFactorAuth) error {
	return nil
}

func (m *mockAuthRepo) Get2FAByUserID(_ context.Context, _ string) (*TwoFactorAuth, error) {
	return nil, Err2FANotEnabled
}

func (m *mockAuthRepo) Enable2FA(_ context.Context, _ string) error {
	return nil
}

func (m *mockAuthRepo) Delete2FA(_ context.Context, _ string) error {
	return nil
}

func (m *mockAuthRepo) StoreRecoveryCodes(_ context.Context, _ string, _ []string) error {
	return nil
}

// --- Helpers ---

func seedUser(repo *mockUserRepo) *user.User {
	uid := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	u := &user.User{
		Id:           uid,
		Email:        "test@example.com",
		PasswordHash: string(hash),
		Role:         "customer",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.users[u.Email] = u
	return u
}

// --- Tests ---

func TestLogin_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	authRepo := newMockAuthRepo()
	seedUser(userRepo)
	svc := NewService(userRepo, authRepo, testSecret)

	resp, err := svc.Login(context.Background(), LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Len(t, authRepo.tokens, 1, "refresh token should be stored")
}

func TestLogin_WrongPassword(t *testing.T) {
	userRepo := newMockUserRepo()
	authRepo := newMockAuthRepo()
	seedUser(userRepo)
	svc := NewService(userRepo, authRepo, testSecret)

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "test@example.com",
		Password: "wrong-password",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := newMockUserRepo()
	authRepo := newMockAuthRepo()
	svc := NewService(userRepo, authRepo, testSecret)

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "nobody@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestLogin_InvalidInput(t *testing.T) {
	svc := NewService(newMockUserRepo(), newMockAuthRepo(), testSecret)

	_, err := svc.Login(context.Background(), LoginRequest{Email: "", Password: ""})
	assert.Error(t, err)
}

func TestRefreshTokens_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	authRepo := newMockAuthRepo()
	u := seedUser(userRepo)
	svc := NewService(userRepo, authRepo, testSecret)

	// Login first to get a valid refresh token.
	loginResp, err := svc.Login(context.Background(), LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	require.NoError(t, err)

	// Now refresh.
	refreshResp, err := svc.RefreshTokens(context.Background(), RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	require.NoError(t, err)

	assert.NotEmpty(t, refreshResp.AccessToken)
	assert.NotEmpty(t, refreshResp.RefreshToken)
	assert.NotEqual(t, loginResp.RefreshToken, refreshResp.RefreshToken, "must issue a new refresh token")
	assert.NotEqual(t, loginResp.AccessToken, refreshResp.AccessToken, "must issue a new access token")

	// Old token should be marked as used.
	oldToken := authRepo.tokens[loginResp.RefreshToken]
	assert.True(t, oldToken.Used)

	// New token should be stored.
	_, exists := authRepo.tokens[refreshResp.RefreshToken]
	assert.True(t, exists)
	_ = u
}

func TestRefreshTokens_ReplayDetection(t *testing.T) {
	userRepo := newMockUserRepo()
	authRepo := newMockAuthRepo()
	seedUser(userRepo)
	svc := NewService(userRepo, authRepo, testSecret)

	loginResp, err := svc.Login(context.Background(), LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	require.NoError(t, err)

	// First refresh should succeed.
	_, err = svc.RefreshTokens(context.Background(), RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	require.NoError(t, err)

	// Second use of the SAME token = replay attack.
	// Should revoke ALL tokens for the user.
	_, err = svc.RefreshTokens(context.Background(), RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	assert.ErrorIs(t, err, ErrTokenUsed)
	assert.NotEmpty(t, authRepo.revokedAllCalls, "should have revoked all user tokens")
}

func TestRefreshTokens_RevokedToken(t *testing.T) {
	userRepo := newMockUserRepo()
	authRepo := newMockAuthRepo()
	seedUser(userRepo)
	svc := NewService(userRepo, authRepo, testSecret)

	loginResp, err := svc.Login(context.Background(), LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	require.NoError(t, err)

	// Manually revoke the token.
	authRepo.tokens[loginResp.RefreshToken].Revoked = true

	_, err = svc.RefreshTokens(context.Background(), RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	assert.ErrorIs(t, err, ErrTokenRevoked)
}

func TestRefreshTokens_ExpiredToken(t *testing.T) {
	userRepo := newMockUserRepo()
	authRepo := newMockAuthRepo()
	seedUser(userRepo)
	svc := NewService(userRepo, authRepo, testSecret)

	loginResp, err := svc.Login(context.Background(), LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	require.NoError(t, err)

	// Manually expire the stored token.
	authRepo.tokens[loginResp.RefreshToken].ExpiresAt = time.Now().Add(-1 * time.Hour)

	_, err = svc.RefreshTokens(context.Background(), RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestRefreshTokens_UnknownToken(t *testing.T) {
	svc := NewService(newMockUserRepo(), newMockAuthRepo(), testSecret)

	_, err := svc.RefreshTokens(context.Background(), RefreshRequest{
		RefreshToken: "totally-not-a-real-token",
	})
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestLogout_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	authRepo := newMockAuthRepo()
	u := seedUser(userRepo)
	svc := NewService(userRepo, authRepo, testSecret)

	// Login twice to create multiple tokens.
	_, err := svc.Login(context.Background(), LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	require.NoError(t, err)
	_, err = svc.Login(context.Background(), LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	require.NoError(t, err)

	assert.Len(t, authRepo.tokens, 2)

	err = svc.Logout(context.Background(), u.Id.String())
	require.NoError(t, err)

	// All tokens should be revoked.
	for _, tok := range authRepo.tokens {
		assert.True(t, tok.Revoked, "all tokens should be revoked after logout")
	}
}
