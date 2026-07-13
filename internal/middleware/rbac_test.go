package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/ctxkey"
	"github.com/stretchr/testify/assert"
)

func withRole(req *http.Request, role string) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), ctxkey.Role, role))
}

func TestRequireRole_AllowsListedRole(t *testing.T) {
	called := false
	h := RequireRole(RoleAdmin, RoleSales)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, withRole(httptest.NewRequest(http.MethodGet, "/", nil), RoleSales))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}

func TestRequireRole_RejectsUnlistedRole(t *testing.T) {
	h := RequireRole(RoleAdmin, RoleSales)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not run")
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, withRole(httptest.NewRequest(http.MethodGet, "/", nil), RoleSupport))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireStaff2FA_BlocksStaffWithout2FA(t *testing.T) {
	check := TwoFactorChecker(func(context.Context, string) (bool, error) { return false, nil })
	h := RequireStaff2FA(check)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not run")
	}))

	req := withRole(httptest.NewRequest(http.MethodGet, "/", nil), RoleAdmin)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.UserID, "u1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), "2fa_required")
}

func TestRequireStaff2FA_AllowsStaffWith2FA(t *testing.T) {
	called := false
	check := TwoFactorChecker(func(context.Context, string) (bool, error) { return true, nil })
	h := RequireStaff2FA(check)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, withRole(httptest.NewRequest(http.MethodGet, "/", nil), RoleAdmin))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}

func TestRequireStaff2FA_IgnoresNonStaff(t *testing.T) {
	// A customer should pass the 2FA gate untouched (RBAC blocks them elsewhere);
	// the checker must not even be consulted.
	called := false
	check := TwoFactorChecker(func(context.Context, string) (bool, error) {
		t.Fatal("checker should not run for non-staff")
		return false, nil
	})
	h := RequireStaff2FA(check)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, withRole(httptest.NewRequest(http.MethodGet, "/", nil), RoleCustomer))

	assert.True(t, called)
}

func TestRequireStaff2FA_CheckerError(t *testing.T) {
	check := TwoFactorChecker(func(context.Context, string) (bool, error) {
		return false, errors.New("db down")
	})
	h := RequireStaff2FA(check)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not run")
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, withRole(httptest.NewRequest(http.MethodGet, "/", nil), RoleSupport))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
