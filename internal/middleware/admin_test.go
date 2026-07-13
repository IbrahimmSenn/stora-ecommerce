package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/ctxkey"
	"github.com/stretchr/testify/assert"
)

func TestIsAdmin_Allowed(t *testing.T) {
	called := false
	handler := IsAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.Role, "admin"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}

func TestIsAdmin_Forbidden_Customer(t *testing.T) {
	handler := IsAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.Role, "customer"))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), "admin access required")
}

func TestIsAdmin_Forbidden_NoRole(t *testing.T) {
	handler := IsAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
