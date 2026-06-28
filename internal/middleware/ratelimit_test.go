package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func reqFromIP(ip string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = ip + ":40000"
	return r
}

func TestRateLimiter_AllowsBurstThenBlocks(t *testing.T) {
	// Effectively no refill, burst of 3.
	rl := NewRateLimiter(0.0001, 3)
	h := rl.Middleware(okHandler())

	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, reqFromIP("1.2.3.4"))
		assert.Equal(t, http.StatusOK, rr.Code, "request %d within burst should pass", i+1)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, reqFromIP("1.2.3.4"))
	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "request beyond burst should be limited")
	assert.NotEmpty(t, rr.Header().Get("Retry-After"))
	assert.Contains(t, rr.Body.String(), "rate_limited")
}

func TestRateLimiter_IsolatesByIP(t *testing.T) {
	rl := NewRateLimiter(0.0001, 1)
	h := rl.Middleware(okHandler())

	// IP A spends its single token.
	rrA1 := httptest.NewRecorder()
	h.ServeHTTP(rrA1, reqFromIP("10.0.0.1"))
	assert.Equal(t, http.StatusOK, rrA1.Code)

	rrA2 := httptest.NewRecorder()
	h.ServeHTTP(rrA2, reqFromIP("10.0.0.1"))
	assert.Equal(t, http.StatusTooManyRequests, rrA2.Code)

	// IP B is unaffected.
	rrB := httptest.NewRecorder()
	h.ServeHTTP(rrB, reqFromIP("10.0.0.2"))
	assert.Equal(t, http.StatusOK, rrB.Code, "a different IP keeps its own bucket")
}
