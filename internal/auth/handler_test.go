package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/user"
)

// --- API integration tests for auth endpoints ---

func setupAuthTestRouter() (*Handler, *mockUserRepo, *mockAuthRepo) {
	userRepo := newMockUserRepo()
	authRepo := newMockAuthRepo()

	uid := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	userRepo.users["test@example.com"] = &user.User{
		Id: uid, Email: "test@example.com", PasswordHash: string(hash),
		Role: "customer", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	svc := NewService(userRepo, authRepo, testSecret)
	handler := NewHandler(svc)
	return handler, userRepo, authRepo
}

func doPost(handler http.HandlerFunc, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// --- Login endpoint integration tests ---

func TestLoginEndpoint_Success(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	rr := doPost(h.Login, "/api/v1/auth/login", LoginRequest{
		Email: "test@example.com", Password: "password123",
	})

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp LoginResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
}

func TestLoginEndpoint_WrongPassword(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	rr := doPost(h.Login, "/api/v1/auth/login", LoginRequest{
		Email: "test@example.com", Password: "wrong",
	})

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid email or password")
}

func TestLoginEndpoint_NonexistentUser(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	rr := doPost(h.Login, "/api/v1/auth/login", LoginRequest{
		Email: "nobody@example.com", Password: "password123",
	})

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestLoginEndpoint_InvalidJSON(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestLoginEndpoint_EmptyBody(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// --- Refresh endpoint integration tests ---

func TestRefreshEndpoint_Success(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	// Login first.
	loginRR := doPost(h.Login, "/api/v1/auth/login", LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	var loginResp LoginResponse
	require.NoError(t, json.Unmarshal(loginRR.Body.Bytes(), &loginResp))

	// Refresh.
	refreshRR := doPost(h.Refresh, "/api/v1/auth/refresh", RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})

	assert.Equal(t, http.StatusOK, refreshRR.Code)

	var refreshResp LoginResponse
	require.NoError(t, json.Unmarshal(refreshRR.Body.Bytes(), &refreshResp))
	assert.NotEmpty(t, refreshResp.AccessToken)
	assert.NotEqual(t, loginResp.RefreshToken, refreshResp.RefreshToken, "should issue new refresh token")
}

func TestRefreshEndpoint_ReplayAttack(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	loginRR := doPost(h.Login, "/api/v1/auth/login", LoginRequest{
		Email: "test@example.com", Password: "password123",
	})
	var loginResp LoginResponse
	require.NoError(t, json.Unmarshal(loginRR.Body.Bytes(), &loginResp))

	// First refresh succeeds.
	rr1 := doPost(h.Refresh, "/api/v1/auth/refresh", RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	assert.Equal(t, http.StatusOK, rr1.Code)

	// Second use of same token = replay attack, should fail.
	rr2 := doPost(h.Refresh, "/api/v1/auth/refresh", RefreshRequest{
		RefreshToken: loginResp.RefreshToken,
	})
	assert.Equal(t, http.StatusUnauthorized, rr2.Code)
	assert.Contains(t, rr2.Body.String(), "already used")
}

func TestRefreshEndpoint_InvalidToken(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	rr := doPost(h.Refresh, "/api/v1/auth/refresh", RefreshRequest{
		RefreshToken: "completely-fake-token",
	})

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRefreshEndpoint_EmptyToken(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	// Empty body field AND no refresh_token cookie present — semantically
	// "no credentials", so 401 (was 400 before cookie support landed).
	rr := doPost(h.Refresh, "/api/v1/auth/refresh", RefreshRequest{
		RefreshToken: "",
	})

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// --- Logout endpoint integration tests ---

func TestLogoutEndpoint_Success(t *testing.T) {
	h, userRepo, _ := setupAuthTestRouter()

	// Get the user ID for the context.
	u := userRepo.users["test@example.com"]

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	// Simulate the auth middleware having set the context.
	ctx := context.WithValue(req.Context(), ctxkey.UserID, u.Id.String())
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.Logout(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "logged out successfully")
}

func TestLogoutEndpoint_NoAuth(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rr := httptest.NewRecorder()
	h.Logout(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

