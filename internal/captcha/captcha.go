// captcha.go ��� reCAPTCHA v3 server-side verification. Skippable for local dev.
package captcha

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const verifyURL = "https://www.google.com/recaptcha/api/siteverify"

// Verifier validates reCAPTCHA v3 tokens.
type Verifier struct {
	secretKey  string
	skip       bool
	threshold  float64
	httpClient *http.Client
}

// NewVerifier creates a reCAPTCHA v3 verifier.
// If skip is true, all tokens are accepted (for local development).
func NewVerifier(secretKey string, skip bool) *Verifier {
	return &Verifier{
		secretKey: secretKey,
		skip:      skip,
		threshold: 0.5,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type verifyResponse struct {
	Success    bool     `json:"success"`
	Score      float64  `json:"score"`
	Action     string   `json:"action"`
	ErrorCodes []string `json:"error-codes"`
}

// Verify checks a reCAPTCHA v3 token. Returns nil if valid.
func (v *Verifier) Verify(token string) error {
	if v.skip {
		return nil
	}

	if token == "" {
		return fmt.Errorf("captcha token is required")
	}

	resp, err := v.httpClient.PostForm(verifyURL, url.Values{
		"secret":   {v.secretKey},
		"response": {token},
	})
	if err != nil {
		return fmt.Errorf("captcha verification request failed: %w", err)
	}
	defer resp.Body.Close()

	var result verifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("captcha verification decode failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("captcha verification failed: %v", result.ErrorCodes)
	}

	if result.Score < v.threshold {
		return fmt.Errorf("captcha score too low: %.2f", result.Score)
	}

	return nil
}
