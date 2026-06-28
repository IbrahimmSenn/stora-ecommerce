package middleware

import (
	"net/http"
	"strings"
)

// ScopePath wraps a middleware so it only runs for request paths under prefix
// and not listed in exempt. Everything else bypasses mw entirely. Used to keep
// the API rate limiter off static assets, /media images, and the Stripe webhook
// (whose retries must not be throttled).
func ScopePath(mw func(http.Handler) http.Handler, prefix string, exempt ...string) func(http.Handler) http.Handler {
	exemptSet := make(map[string]struct{}, len(exempt))
	for _, p := range exempt {
		exemptSet[p] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		wrapped := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, skip := exemptSet[r.URL.Path]; skip || !strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}
			wrapped.ServeHTTP(w, r)
		})
	}
}
