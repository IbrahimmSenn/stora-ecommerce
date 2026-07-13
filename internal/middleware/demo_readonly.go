// demo_readonly.go — blocks admin mutations on public demo deployments.
package middleware

import (
	"net/http"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/response"
)

// DemoReadOnly rejects mutating requests when the deployment runs as a public
// demo, so visitors can browse the admin dashboard without changing data.
// Reads pass through. When disabled it is a no-op.
func DemoReadOnly(enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !enabled {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				response.ErrorWithCode(w, http.StatusForbidden, "demo_readonly",
					"Demo mode — admin changes are disabled.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
