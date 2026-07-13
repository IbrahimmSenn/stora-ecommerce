package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func resolvedIP(t *testing.T, remoteAddr string, headers map[string]string, trusted []string) string {
	t.Helper()
	var got string
	h := RealIP(trusted)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.RemoteAddr
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	h.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestRealIP_IgnoresHeadersFromUntrustedPeer(t *testing.T) {
	// A direct (untrusted) client tries to spoof its IP — headers ignored.
	got := resolvedIP(t, "203.0.113.7:5000", map[string]string{
		"X-Forwarded-For": "1.2.3.4",
		"X-Real-IP":       "5.6.7.8",
		"True-Client-IP":  "9.9.9.9",
	}, []string{"10.0.0.0/8"})
	assert.Equal(t, "203.0.113.7", got)
}

func TestRealIP_HonoursXFFFromTrustedProxy(t *testing.T) {
	got := resolvedIP(t, "10.1.2.3:5000", map[string]string{
		"X-Forwarded-For": "198.51.100.42",
	}, []string{"10.0.0.0/8"})
	assert.Equal(t, "198.51.100.42", got)
}

func TestRealIP_XFFRightmostUntrustedWins(t *testing.T) {
	// Client prepends a fake hop; Caddy appends the real one. The rightmost
	// non-proxy entry is the real client.
	got := resolvedIP(t, "10.1.2.3:5000", map[string]string{
		"X-Forwarded-For": "1.1.1.1, 198.51.100.42",
	}, []string{"10.0.0.0/8"})
	assert.Equal(t, "198.51.100.42", got)
}

func TestRealIP_FallsBackToXRealIP(t *testing.T) {
	got := resolvedIP(t, "10.1.2.3:5000", map[string]string{
		"X-Real-IP": "198.51.100.9",
	}, []string{"10.0.0.0/8"})
	assert.Equal(t, "198.51.100.9", got)
}

func TestRealIP_NoHeadersUsesPeer(t *testing.T) {
	got := resolvedIP(t, "10.1.2.3:5000", nil, []string{"10.0.0.0/8"})
	assert.Equal(t, "10.1.2.3", got)
}
