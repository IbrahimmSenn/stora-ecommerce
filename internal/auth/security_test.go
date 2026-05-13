package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Security tests for authentication endpoints.
// These verify the system properly handles malicious input,
// injection attacks, and authentication bypass attempts.

// --- SQL injection tests ---

func TestSecurity_SQLInjection_Login(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	injections := []string{
		"' OR '1'='1",
		"'; DROP TABLE users; --",
		"' UNION SELECT * FROM users --",
		"admin'--",
		"1; DELETE FROM users",
		"' OR 1=1 --",
	}

	for _, payload := range injections {
		t.Run(payload, func(t *testing.T) {
			rr := doPost(h.Login, "/api/v1/auth/login", LoginRequest{
				Email:    payload,
				Password: payload,
			})
			// Should be rejected by validation or return unauthorized — never 200 or 500.
			assert.True(t, rr.Code == http.StatusBadRequest || rr.Code == http.StatusUnauthorized,
				"SQL injection should be rejected, got %d for payload: %s", rr.Code, payload)
		})
	}
}

func TestSecurity_SQLInjection_Refresh(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	injections := []string{
		"' OR '1'='1",
		"'; DROP TABLE refresh_tokens; --",
		"' UNION SELECT token FROM refresh_tokens --",
	}

	for _, payload := range injections {
		t.Run(payload, func(t *testing.T) {
			rr := doPost(h.Refresh, "/api/v1/auth/refresh", RefreshRequest{
				RefreshToken: payload,
			})
			assert.Equal(t, http.StatusUnauthorized, rr.Code,
				"SQL injection in refresh token should return 401")
		})
	}
}

// --- XSS payload tests ---

func TestSecurity_XSS_Login(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	xssPayloads := []string{
		`<script>alert('xss')</script>`,
		`"><img src=x onerror=alert(1)>`,
		`javascript:alert(1)`,
		`<svg onload=alert(1)>`,
	}

	for _, payload := range xssPayloads {
		t.Run(payload, func(t *testing.T) {
			rr := doPost(h.Login, "/api/v1/auth/login", LoginRequest{
				Email:    payload,
				Password: "password123",
			})
			// Should not reflect the XSS payload back in the response.
			assert.NotContains(t, rr.Body.String(), "<script>",
				"response should not contain raw script tags")
			assert.NotContains(t, rr.Body.String(), "onerror=",
				"response should not contain event handlers")
		})
	}
}

// --- Malformed JSON tests ---

func TestSecurity_MalformedJSON_Login(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	payloads := []string{
		``,
		`{`,
		`{invalid`,
		`{"email": "test@example.com"`,
		`null`,
		`[]`,
		`"string"`,
		`12345`,
	}

	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
				bytes.NewReader([]byte(payload)))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			h.Login(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code,
				"malformed JSON should return 400")
		})
	}
}

func TestSecurity_MalformedJSON_Refresh(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	// Each payload pairs with the status we expect after cookie-fallback
	// support was added: truly malformed JSON still returns 400, but valid
	// JSON that simply carries no token (empty body, `null`) now returns
	// 401 because the cookie fallback also yields no token.
	cases := []struct {
		payload string
		status  int
	}{
		{``, http.StatusUnauthorized},
		{`{`, http.StatusBadRequest},
		{`null`, http.StatusUnauthorized},
		{`[]`, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.payload, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh",
				bytes.NewReader([]byte(tc.payload)))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			h.Refresh(rr, req)

			assert.Equal(t, tc.status, rr.Code)
		})
	}
}

// --- Oversized payload tests ---

func TestSecurity_OversizedPayload_Login(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	// 1MB payload.
	huge := `{"email":"` + strings.Repeat("a", 1_000_000) + `@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewReader([]byte(huge)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	// Should not crash — either 400 or 401 is acceptable.
	assert.True(t, rr.Code >= 400 && rr.Code < 500,
		"oversized payload should return client error, got %d", rr.Code)
}

// --- Unknown fields rejection ---

func TestSecurity_UnknownFields_Login(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	// DisallowUnknownFields should reject this.
	payload := `{"email":"test@example.com","password":"password123","admin":true,"role":"admin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code,
		"unknown fields like 'admin' or 'role' should be rejected")
}

// --- Token tampering tests ---

func TestSecurity_TamperedToken_Refresh(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	// A real-looking but tampered JWT (modified payload).
	tampered := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	rr := doPost(h.Refresh, "/api/v1/auth/refresh", RefreshRequest{
		RefreshToken: tampered,
	})

	assert.Equal(t, http.StatusUnauthorized, rr.Code,
		"tampered JWT should be rejected")
}

// --- Timing / enumeration resistance ---

func TestSecurity_UserEnumeration_Login(t *testing.T) {
	h, _, _ := setupAuthTestRouter()

	// Both existing and non-existing users should return the same error message
	// to prevent user enumeration attacks.
	rrExisting := doPost(h.Login, "/api/v1/auth/login", LoginRequest{
		Email: "test@example.com", Password: "wrongpassword",
	})
	rrNonExisting := doPost(h.Login, "/api/v1/auth/login", LoginRequest{
		Email: "nonexistent@example.com", Password: "wrongpassword",
	})

	assert.Equal(t, rrExisting.Code, rrNonExisting.Code,
		"same status code for existing and non-existing users")
	assert.Equal(t, rrExisting.Body.String(), rrNonExisting.Body.String(),
		"same error message to prevent user enumeration")
}
