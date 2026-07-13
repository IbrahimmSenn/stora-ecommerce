// rbac.go — role-based access control and admin 2FA enforcement.
package middleware

import (
	"context"
	"net/http"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/ctxkey"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/response"
)

// Roles. Staff roles carry privileged access; customer is the default.
const (
	RoleAdmin    = "admin"
	RoleSupport  = "support"
	RoleSales    = "sales"
	RoleCustomer = "customer"
)

// IsStaff reports whether a role is one of the privileged staff roles.
func IsStaff(role string) bool {
	return role == RoleAdmin || role == RoleSupport || role == RoleSales
}

// RequireRole rejects requests whose authenticated role is not in allowed.
// Must run after Auth. This is the RBAC enforcement point — apply the least
// set of roles each route group actually needs.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	allow := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allow[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(ctxkey.Role).(string)
			if _, ok := allow[role]; !ok {
				response.Error(w, http.StatusForbidden, "you do not have permission to perform this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// TwoFactorChecker reports whether the given user has 2FA enabled. Injected to
// keep middleware decoupled from the auth package.
type TwoFactorChecker func(ctx context.Context, userID string) (bool, error)

// RequireStaff2FA blocks staff users who have not enabled 2FA from reaching the
// guarded routes, returning a stable "2fa_required" code so the frontend can
// route them to the setup flow. Non-staff requests pass through untouched
// (they are rejected earlier by RequireRole on admin routes). Must run after Auth.
func RequireStaff2FA(check TwoFactorChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(ctxkey.Role).(string)
			if IsStaff(role) {
				userID, _ := r.Context().Value(ctxkey.UserID).(string)
				enabled, err := check(r.Context(), userID)
				if err != nil {
					response.Error(w, http.StatusInternalServerError, "could not verify two-factor status")
					return
				}
				if !enabled {
					response.ErrorWithCode(w, http.StatusForbidden, "2fa_required",
						"two-factor authentication must be enabled on staff accounts before accessing the admin area")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
