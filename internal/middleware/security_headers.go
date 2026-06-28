package middleware

import "net/http"

// SecurityHeaders sets baseline response headers that harden the browser against
// common attacks (clickjacking, MIME sniffing, referrer leakage). hsts enables
// Strict-Transport-Security — only turn it on in production behind HTTPS, since
// it forces browsers to refuse plain HTTP for the whole domain.
//
// The CSP is scoped for an SPA served same-origin: scripts/styles/fonts from
// self, images from self plus data/blob URIs. Stripe.js is allowed explicitly
// (script + the Elements iframes from js.stripe.com, API calls to api.stripe.com)
// since the checkout depends on it — without these directives Elements won't load.
func SecurityHeaders(hsts bool) func(http.Handler) http.Handler {
	const csp = "default-src 'self'; " +
		"script-src 'self' https://js.stripe.com; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: blob:; " +
		"font-src 'self' data:; " +
		"connect-src 'self' https://api.stripe.com; " +
		"frame-src https://js.stripe.com https://hooks.stripe.com; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Content-Security-Policy", csp)
			if hsts {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}
