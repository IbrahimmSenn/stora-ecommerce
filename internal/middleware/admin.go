// admin.go — middleware that rejects non-admin users.
package middleware

import (
	"net/http"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

// IsAdmin rejects requests where the authenticated user's role is not "admin".
// Must be placed after the Auth middleware in the middleware chain.
func IsAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(ctxkey.Role).(string)
		if role != "admin" {
			response.Error(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
