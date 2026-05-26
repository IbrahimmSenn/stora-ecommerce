// geocoder.go — shipping address verification via an external geocoder.
//
// The default implementation calls OpenStreetMap's Nominatim service. It's
// free, key-less, and acceptable for a school demo — production traffic
// should use a paid provider (SmartyStreets, Loqate, etc.) per Nominatim's
// 1-rps usage policy.
package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Geocoder verifies that a shipping address resolves to a real place.
//
// Returns nil when the address is verified.
// Returns ErrAddressNotVerifiable when the provider returns no match.
// Returns ErrAddressVerificationUnavailable for transient failures (network,
// 5xx) so the caller can distinguish "address is bad" from "we couldn't ask".
type Geocoder interface {
	VerifyAddress(ctx context.Context, addr CheckoutAddressRequest) error
}

// PassthroughGeocoder accepts every address. Used in tests and as a safe
// default if a Geocoder isn't wired up.
type PassthroughGeocoder struct{}

func (PassthroughGeocoder) VerifyAddress(_ context.Context, _ CheckoutAddressRequest) error {
	return nil
}

// nominatimClient is the production Geocoder. Nominatim requires a
// descriptive User-Agent identifying the application — requests without one
// get 403'd per the OSM usage policy.
type nominatimClient struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
}

// NewNominatimGeocoder builds a Geocoder backed by Nominatim. baseURL should
// be the root (no trailing /search), e.g. "https://nominatim.openstreetmap.org".
// userAgent is required and must identify the app + a contact address.
func NewNominatimGeocoder(baseURL, userAgent string) Geocoder {
	return &nominatimClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
		userAgent:  userAgent,
	}
}

type nominatimResult struct {
	PlaceID int `json:"place_id"`
	// We only need to know whether the array was empty; the rest of the
	// response is ignored on purpose. Storing place_id makes failures easier
	// to read in logs if we ever need to debug.
}

func (n *nominatimClient) VerifyAddress(ctx context.Context, addr CheckoutAddressRequest) error {
	q := url.Values{}
	q.Set("format", "jsonv2")
	q.Set("addressdetails", "1")
	q.Set("limit", "1")
	q.Set("street", joinStreet(addr.Line1, addr.Line2))
	q.Set("city", addr.City)
	q.Set("state", addr.Region)
	q.Set("postalcode", addr.PostalCode)
	q.Set("country", addr.Country)

	endpoint := n.baseURL + "/search?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("%w: build request: %v", ErrAddressVerificationUnavailable, err)
	}
	req.Header.Set("User-Agent", n.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAddressVerificationUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: upstream status %d", ErrAddressVerificationUnavailable, resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("%w: rate limited", ErrAddressVerificationUnavailable)
	}
	if resp.StatusCode >= 400 {
		// 4xx other than 429 means our request was malformed in a way the
		// provider rejected — treat as unavailable rather than "not found"
		// so we don't penalise the user for our bug.
		return fmt.Errorf("%w: upstream status %d", ErrAddressVerificationUnavailable, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: read body: %v", ErrAddressVerificationUnavailable, err)
	}
	var results []nominatimResult
	if err := json.Unmarshal(body, &results); err != nil {
		return fmt.Errorf("%w: decode body: %v", ErrAddressVerificationUnavailable, err)
	}
	if len(results) == 0 {
		return ErrAddressNotVerifiable
	}
	return nil
}

// joinStreet folds Line2 onto Line1 with a comma. Nominatim's `street`
// parameter wants the whole street part as one string.
func joinStreet(line1, line2 string) string {
	if line2 == "" {
		return line1
	}
	return line1 + ", " + line2
}

// errorCodeFor maps a geocoder error to the structured code returned to the
// frontend. Used by the orders handler so the React side can key off a
// stable string instead of string-matching the user-facing message.
func errorCodeFor(err error) string {
	switch {
	case errors.Is(err, ErrAddressNotVerifiable):
		return "address_unverified"
	case errors.Is(err, ErrAddressVerificationUnavailable):
		return "address_verification_unavailable"
	default:
		return ""
	}
}
