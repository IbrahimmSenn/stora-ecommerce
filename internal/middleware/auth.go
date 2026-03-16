package middleware

import (
	"context"
	"net/http"
	"strings"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

// TokenClaims holds the decoded claims from a validated JWT.
type TokenClaims struct {
	UserID string
	Email  string
	Role   string
}

// TokenValidator is a function that validates a JWT string and returns the claims.
// This avoids a direct import of the auth package, preventing import cycles.
type TokenValidator func(tokenString string) (*TokenClaims, error)

// Auth returns middleware that validates the Bearer token in the Authorization
// header and injects the user's claims into the request context.
func Auth(validate TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				response.Error(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}

			claims, err := validate(parts[1])
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), ctxkey.UserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxkey.Email, claims.Email)
			ctx = context.WithValue(ctx, ctxkey.Role, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
