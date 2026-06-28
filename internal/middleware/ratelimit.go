// ratelimit.go — per-client token-bucket rate limiting.
//
// Each client IP gets its own bucket (golang.org/x/time/rate). Buckets are
// kept in a map and swept periodically so idle clients don't leak memory.
// Apply a strict limiter to auth/sensitive routes and a loose one globally.
package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

type clientBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter holds one token bucket per client IP.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientBucket
	rps     rate.Limit
	burst   int
	ttl     time.Duration
}

// NewRateLimiter builds a limiter allowing `rps` requests/second per IP with a
// burst of `burst`. Idle client buckets are evicted after they go untouched
// for longer than the sweep TTL.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientBucket),
		rps:     rate.Limit(rps),
		burst:   burst,
		ttl:     10 * time.Minute,
	}
	go rl.sweep()
	return rl
}

func (rl *RateLimiter) bucket(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if c, ok := rl.clients[ip]; ok {
		c.lastSeen = time.Now()
		return c.limiter
	}
	lim := rate.NewLimiter(rl.rps, rl.burst)
	rl.clients[ip] = &clientBucket{limiter: lim, lastSeen: time.Now()}
	return lim
}

func (rl *RateLimiter) sweep() {
	t := time.NewTicker(rl.ttl)
	defer t.Stop()
	for range t.C {
		rl.mu.Lock()
		for ip, c := range rl.clients {
			if time.Since(c.lastSeen) > rl.ttl {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware rejects requests from a client that has exhausted its bucket with
// 429 and a Retry-After header.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.bucket(clientIP(r)).Allow() {
			w.Header().Set("Retry-After", "1")
			response.ErrorWithCode(w, http.StatusTooManyRequests, "rate_limited",
				"too many requests — please slow down and try again in a moment")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the host portion of RemoteAddr. RealIP middleware upstream
// already resolves X-Forwarded-For / X-Real-IP into RemoteAddr.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
