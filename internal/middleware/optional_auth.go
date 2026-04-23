package middleware

import (
	"context"
	"net/http"
	"strings"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
)

// OptionalAuth validates a Bearer token if present and injects claims into the
// context. Unlike Auth, it does not reject requests that have no token or an
// invalid token — it just passes through. Used for routes that work for both
// authenticated and guest users (e.g. cart).
func OptionalAuth(validate TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := validate(parts[1])
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), ctxkey.UserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxkey.Email, claims.Email)
			ctx = context.WithValue(ctx, ctxkey.Role, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
