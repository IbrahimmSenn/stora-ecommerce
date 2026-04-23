package middleware

import (
	"net/http"

	"github.com/google/uuid"
)

const GuestSessionCookie = "guest_session_id"

// GuestSession ensures an HTTP-only cookie holding a UUID exists on the
// request. If missing or malformed, a new UUID is generated and set on the
// response. Authenticated users are skipped — their cart is tied to user_id.
// Must run after OptionalAuth so the context check works.
func GuestSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(GuestSessionCookie); err == nil {
			if _, err := uuid.Parse(c.Value); err == nil {
				next.ServeHTTP(w, r)
				return
			}
		}

		sessionID := uuid.New().String()
		http.SetCookie(w, &http.Cookie{
			Name:     GuestSessionCookie,
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   60 * 60 * 24 * 30, // 30 days
		})
		// Also make it available to the current request's handler.
		r.AddCookie(&http.Cookie{Name: GuestSessionCookie, Value: sessionID})
		next.ServeHTTP(w, r)
	})
}
