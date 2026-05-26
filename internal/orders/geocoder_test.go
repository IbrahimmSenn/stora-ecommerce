package orders

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func validAddress() CheckoutAddressRequest {
	return CheckoutAddressRequest{
		RecipientName: "Buyer",
		Line1:         "1 Infinite Loop",
		City:          "Cupertino",
		Region:        "CA",
		PostalCode:    "95014",
		Country:       "US",
	}
}

func TestNominatim_Hit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("expected User-Agent header")
		}
		if r.URL.Query().Get("street") != "1 Infinite Loop" {
			t.Errorf("street param missing/wrong: %q", r.URL.Query().Get("street"))
		}
		io.WriteString(w, `[{"place_id":12345}]`)
	}))
	defer srv.Close()

	g := NewNominatimGeocoder(srv.URL, "test/1.0 (test@example.com)")
	if err := g.VerifyAddress(context.Background(), validAddress()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestNominatim_NoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[]`)
	}))
	defer srv.Close()

	g := NewNominatimGeocoder(srv.URL, "test/1.0")
	err := g.VerifyAddress(context.Background(), validAddress())
	if !errors.Is(err, ErrAddressNotVerifiable) {
		t.Fatalf("expected ErrAddressNotVerifiable, got %v", err)
	}
}

func TestNominatim_5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	g := NewNominatimGeocoder(srv.URL, "test/1.0")
	err := g.VerifyAddress(context.Background(), validAddress())
	if !errors.Is(err, ErrAddressVerificationUnavailable) {
		t.Fatalf("expected ErrAddressVerificationUnavailable, got %v", err)
	}
}

func TestNominatim_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	g := NewNominatimGeocoder(srv.URL, "test/1.0")
	err := g.VerifyAddress(context.Background(), validAddress())
	if !errors.Is(err, ErrAddressVerificationUnavailable) {
		t.Fatalf("expected ErrAddressVerificationUnavailable, got %v", err)
	}
}

func TestNominatim_NetworkError(t *testing.T) {
	// Point at a closed port to force a connection refused.
	g := NewNominatimGeocoder("http://127.0.0.1:1", "test/1.0")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := g.VerifyAddress(ctx, validAddress())
	if !errors.Is(err, ErrAddressVerificationUnavailable) {
		t.Fatalf("expected ErrAddressVerificationUnavailable, got %v", err)
	}
}

func TestNominatim_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `not json`)
	}))
	defer srv.Close()

	g := NewNominatimGeocoder(srv.URL, "test/1.0")
	err := g.VerifyAddress(context.Background(), validAddress())
	if !errors.Is(err, ErrAddressVerificationUnavailable) {
		t.Fatalf("expected ErrAddressVerificationUnavailable, got %v", err)
	}
}

func TestErrorCodeFor(t *testing.T) {
	if got := errorCodeFor(ErrAddressNotVerifiable); got != "address_unverified" {
		t.Errorf("not verifiable → %q", got)
	}
	if got := errorCodeFor(ErrAddressVerificationUnavailable); got != "address_verification_unavailable" {
		t.Errorf("unavailable → %q", got)
	}
	if got := errorCodeFor(ErrCartEmpty); got != "" {
		t.Errorf("unrelated error should map to empty, got %q", got)
	}
}
