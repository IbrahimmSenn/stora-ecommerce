package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	"github.com/stretchr/testify/assert"
)

func validValidator(token string) (*TokenClaims, error) {
	if token == "valid-token" {
		return &TokenClaims{UserID: "user-123", Email: "test@example.com", Role: "user"}, nil
	}
	return nil, errors.New("invalid token")
}

func TestAuth_MissingHeader(t *testing.T) {
	handler := Auth(validValidator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing authorization header")
}

func TestAuth_InvalidFormat(t *testing.T) {
	tests := []string{
		"InvalidFormat",
		"Basic dXNlcjpwYXNz",
		"Bearer",
		"bearer",
	}

	for _, header := range tests {
		t.Run(header, func(t *testing.T) {
			handler := Auth(validValidator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("handler should not be called")
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", header)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	handler := Auth(validValidator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid or expired token")
}

func TestAuth_ValidToken(t *testing.T) {
	var capturedUserID, capturedEmail, capturedRole string

	handler := Auth(validValidator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID, _ = r.Context().Value(ctxkey.UserID).(string)
		capturedEmail, _ = r.Context().Value(ctxkey.Email).(string)
		capturedRole, _ = r.Context().Value(ctxkey.Role).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "user-123", capturedUserID)
	assert.Equal(t, "test@example.com", capturedEmail)
	assert.Equal(t, "user", capturedRole)
}

func TestAuth_BearerCaseInsensitive(t *testing.T) {
	called := false
	handler := Auth(validValidator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "BEARER valid-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}
