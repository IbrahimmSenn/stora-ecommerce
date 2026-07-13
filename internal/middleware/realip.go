// realip.go — trusted-proxy-aware client IP resolution.
//
// chi's stock RealIP believes X-Forwarded-For / X-Real-IP / True-Client-IP
// from any caller, so a client can send a fresh spoofed IP per request and mint
// a new rate-limit bucket each time (CWE-307). This version only honours those
// headers when the immediate connection comes from a trusted proxy CIDR;
// otherwise the real connection address is used. That keeps the per-IP rate
// limiter and access logs honest behind Caddy while remaining safe if the app
// is ever exposed directly.
package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"
)

// RealIP returns middleware that rewrites r.RemoteAddr to the resolved client
// IP. trustedCIDRs are the proxy ranges whose forwarding headers are believed;
// unparseable entries are logged and skipped.
func RealIP(trustedCIDRs []string) func(http.Handler) http.Handler {
	trusted := make([]*net.IPNet, 0, len(trustedCIDRs))
	for _, c := range trustedCIDRs {
		_, netw, err := net.ParseCIDR(strings.TrimSpace(c))
		if err != nil {
			log.Printf("realip: ignoring invalid trusted proxy CIDR %q: %v", c, err)
			continue
		}
		trusted = append(trusted, netw)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ip := clientIPFromRequest(r, trusted); ip != "" {
				r.RemoteAddr = ip
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isTrusted(ip net.IP, trusted []*net.IPNet) bool {
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// clientIPFromRequest resolves the client IP. If the peer is a trusted proxy it
// honours X-Forwarded-For (walking right-to-left past trusted hops), then
// X-Real-IP; otherwise it returns the peer address unchanged.
func clientIPFromRequest(r *http.Request, trusted []*net.IPNet) string {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = h
	}
	peer := net.ParseIP(host)
	if peer == nil || !isTrusted(peer, trusted) {
		// Untrusted (or unparseable) peer: never believe forwarding headers.
		return host
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		// Rightmost entry not itself a trusted proxy is the real client.
		for i := len(parts) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(parts[i]))
			if ip == nil {
				continue
			}
			if !isTrusted(ip, trusted) {
				return ip.String()
			}
		}
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
		if ip := net.ParseIP(xr); ip != nil {
			return ip.String()
		}
	}
	return host
}
