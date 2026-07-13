// ratelimit.go — per-client token-bucket rate limiting.
//
// The decision is delegated to a limiterStore: an in-memory store (the default,
// one bucket per client IP) or a Redis-backed store shared across instances
// (enabled via REDIS_URL) so horizontal scaling enforces a single global limit.
// Apply a strict limiter to auth/sensitive routes and a loose one to the API.
package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/response"
)

// limiterStore decides whether the client identified by key may proceed. It
// fails open: on a backing-store error it returns true so a cache outage never
// takes the site down.
type limiterStore interface {
	allow(ctx context.Context, key string) bool
}

// RateLimiter is HTTP middleware over a limiterStore.
type RateLimiter struct {
	store    limiterStore
	name     string
	onReject func()
}

// NewRateLimiter builds an in-memory limiter allowing `rps` requests/second per
// IP with the given burst. Single-binary default.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{store: newMemoryStore(rps, burst), name: "general"}
}

// Instrument names the limiter for logs and registers a callback fired on each
// 429 (used to bump the rate-limit rejection metric). Returns the receiver so
// it chains off the constructor.
func (rl *RateLimiter) Instrument(name string, onReject func()) *RateLimiter {
	rl.name = name
	rl.onReject = onReject
	return rl
}

// Middleware rejects a client that has exhausted its bucket with 429 and a
// Retry-After header.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.store.allow(r.Context(), clientIP(r)) {
			slog.Warn("rate_limited", "limiter", rl.name, "ip", clientIP(r), "path", r.URL.Path)
			if rl.onReject != nil {
				rl.onReject()
			}
			w.Header().Set("Retry-After", "1")
			response.ErrorWithCode(w, http.StatusTooManyRequests, "rate_limited",
				"too many requests — please slow down and try again in a moment")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- in-memory store -------------------------------------------------------

type clientBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type memoryStore struct {
	mu      sync.Mutex
	clients map[string]*clientBucket
	rps     rate.Limit
	burst   int
	ttl     time.Duration
}

func newMemoryStore(rps float64, burst int) *memoryStore {
	s := &memoryStore{
		clients: make(map[string]*clientBucket),
		rps:     rate.Limit(rps),
		burst:   burst,
		ttl:     10 * time.Minute,
	}
	go s.sweep()
	return s
}

func (s *memoryStore) allow(_ context.Context, key string) bool {
	return s.bucket(key).Allow()
}

func (s *memoryStore) bucket(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.clients[ip]; ok {
		c.lastSeen = time.Now()
		return c.limiter
	}
	lim := rate.NewLimiter(s.rps, s.burst)
	s.clients[ip] = &clientBucket{limiter: lim, lastSeen: time.Now()}
	return lim
}

func (s *memoryStore) sweep() {
	t := time.NewTicker(s.ttl)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		for ip, c := range s.clients {
			if time.Since(c.lastSeen) > s.ttl {
				delete(s.clients, ip)
			}
		}
		s.mu.Unlock()
	}
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
