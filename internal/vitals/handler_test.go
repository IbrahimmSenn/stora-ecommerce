package vitals

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type recorded struct {
	name  string
	value float64
	calls int
}

func (r *recorded) Observe(name string, value float64) {
	r.name, r.value = name, value
	r.calls++
}

func post(t *testing.T, body string) (*httptest.ResponseRecorder, *recorded) {
	t.Helper()
	rec := &recorded{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/vitals", strings.NewReader(body))
	Handler(rec)(w, r)
	return w, rec
}

func TestHandlerValid(t *testing.T) {
	cases := []struct {
		body string
		name string
	}{
		{`{"name":"LCP","value":1830.5}`, "LCP"},
		{`{"name":"CLS","value":0.04}`, "CLS"},
		{`{"name":"INP","value":0}`, "INP"},
	}
	for _, c := range cases {
		w, rec := post(t, c.body)
		if w.Code != http.StatusNoContent {
			t.Errorf("%s: got %d, want 204", c.body, w.Code)
		}
		if rec.calls != 1 || rec.name != c.name {
			t.Errorf("%s: recorded %q x%d", c.body, rec.name, rec.calls)
		}
	}
}

func TestHandlerRejects(t *testing.T) {
	cases := []struct {
		body string
		code int
	}{
		{`not json`, http.StatusBadRequest},
		{`{"name":"EVIL","value":1}`, http.StatusUnprocessableEntity},
		{`{"name":"LCP","value":-5}`, http.StatusUnprocessableEntity},
		{`{"name":"LCP","value":999999}`, http.StatusUnprocessableEntity},
		{`{"name":"CLS","value":42}`, http.StatusUnprocessableEntity},
		{`{"name":"LCP","value":1,"pad":"` + strings.Repeat("x", 2048) + `"}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		w, rec := post(t, c.body)
		if w.Code != c.code {
			t.Errorf("%.40s: got %d, want %d", c.body, w.Code, c.code)
		}
		if rec.calls != 0 {
			t.Errorf("%.40s: recorder called on invalid input", c.body)
		}
	}
}
