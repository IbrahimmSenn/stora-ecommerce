package product

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Security tests for product catalog endpoints.
// Covers SQL injection, XSS, malformed input, and oversized payloads.

// --- SQL injection via search params ---

func TestSecurity_SQLInjection_Search(t *testing.T) {
	h, _ := setupProductHandler()

	injections := []string{
		"' OR '1'='1",
		"'; DROP TABLE products; --",
		"' UNION SELECT * FROM users --",
		"1; DELETE FROM products",
	}

	for _, payload := range injections {
		t.Run(payload, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/products?q="+url.QueryEscape(payload), nil)
			rr := httptest.NewRecorder()
			h.Search(rr, req)

			// The search should either succeed with empty results or return an error,
			// but never execute the injected SQL.
			assert.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusBadRequest,
				"SQL injection in search should not cause 500, got %d", rr.Code)
		})
	}
}

func TestSecurity_SQLInjection_FilterParams(t *testing.T) {
	h, _ := setupProductHandler()

	// Inject SQL through category_id and brand_id filters.
	params := []string{
		"category_id=' OR '1'='1",
		"brand_id='; DROP TABLE brands; --",
		"min_price=0 OR 1=1",
		"max_price=999999; DELETE FROM products",
	}

	for _, param := range params {
		t.Run(param, func(t *testing.T) {
			// Split the param to properly encode the value.
			parts := strings.SplitN(param, "=", 2)
			req := httptest.NewRequest(http.MethodGet,
				"/api/v1/products?"+parts[0]+"="+url.QueryEscape(parts[1]), nil)
			rr := httptest.NewRecorder()
			h.Search(rr, req)

			// Parameterized queries should prevent injection.
			// These params will be parsed as strings/ints and fail safely.
			assert.True(t, rr.Code < 500,
				"SQL injection in filter params should not cause server error, got %d", rr.Code)
		})
	}
}

// --- XSS in product creation ---

func TestSecurity_XSS_CreateProduct(t *testing.T) {
	h, _ := setupProductHandler()

	xssPayloads := []struct {
		name string
		body CreateProductRequest
	}{
		{
			name: "script_in_name",
			body: CreateProductRequest{
				Name: `<script>alert('xss')</script>`, Price: 1000, WeightG: intPtr(100),
			},
		},
		{
			name: "img_onerror_in_name",
			body: CreateProductRequest{
				Name: `"><img src=x onerror=alert(1)>`, Price: 1000, WeightG: intPtr(100),
			},
		},
		{
			name: "script_in_description",
			body: func() CreateProductRequest {
				desc := `<script>document.cookie</script>`
				return CreateProductRequest{
					Name: "Normal Product", Description: &desc, Price: 1000, WeightG: intPtr(100),
				}
			}(),
		},
	}

	for _, tt := range xssPayloads {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			h.Create(rr, req)

			// The API stores what it receives (JSON API, not HTML),
			// but the response Content-Type must be application/json, not text/html.
			assert.Contains(t, rr.Header().Get("Content-Type"), "application/json",
				"response must be JSON to prevent browser XSS interpretation")
		})
	}
}

// --- Malformed JSON for product endpoints ---

func TestSecurity_MalformedJSON_CreateProduct(t *testing.T) {
	h, _ := setupProductHandler()

	payloads := []string{
		``,
		`{`,
		`not json`,
		`{"name": 12345}`,
		`null`,
		`[]`,
	}

	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products",
				bytes.NewReader([]byte(payload)))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			h.Create(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code,
				"malformed JSON should return 400")
		})
	}
}

func TestSecurity_MalformedJSON_UpdateProduct(t *testing.T) {
	h, repo := setupProductHandler()
	p := seedProduct(repo)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/products/"+p.ID.String(),
		bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", p.ID.String())
	rr := httptest.NewRecorder()
	h.Update(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// --- Oversized payload ---

func TestSecurity_OversizedPayload_CreateProduct(t *testing.T) {
	h, _ := setupProductHandler()

	hugeName := strings.Repeat("A", 1_000_000)
	body := `{"name":"` + hugeName + `","price":1000,"weight_g":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products",
		bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	// Should not crash. Validation may reject the oversized name or it may succeed
	// (DB will enforce varchar(255)), but it must not panic.
	assert.True(t, rr.Code < 500, "oversized payload should not cause server error, got %d", rr.Code)
}

// --- Unknown fields rejection ---

func TestSecurity_UnknownFields_CreateProduct(t *testing.T) {
	h, _ := setupProductHandler()

	// Attempt to set fields that shouldn't be user-controlled.
	payload := `{"name":"Exploit","price":1000,"weight_g":100,"id":"` + uuid.NewString() + `","created_at":"2020-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products",
		bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code,
		"unknown fields like 'id' and 'created_at' should be rejected by DisallowUnknownFields")
}

// --- Invalid UUID in URL path ---

func TestSecurity_InvalidUUID_GetByID(t *testing.T) {
	h, _ := setupProductHandler()

	invalidIDs := []string{
		"not-a-uuid",
		"12345",
		"'; DROP TABLE products; --",
		"<script>alert(1)</script>",
	}

	for _, id := range invalidIDs {
		t.Run(id, func(t *testing.T) {
			// Use a safe path for the request, inject the malicious ID via chi route params only.
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products/test", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()
			h.GetByID(rr, req)

			// Should return 404 or 400, never 500.
			assert.True(t, rr.Code < 500,
				"invalid UUID should not cause server error, got %d for id: %s", rr.Code, id)
		})
	}
}

// --- Negative values ---

func TestSecurity_NegativeValues_CreateProduct(t *testing.T) {
	h, _ := setupProductHandler()

	body := CreateProductRequest{
		Name: "Negative Test", Price: -100, StockQuantity: -5, WeightG: intPtr(-10),
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code,
		"negative price/stock/weight should be rejected by validation")
}
