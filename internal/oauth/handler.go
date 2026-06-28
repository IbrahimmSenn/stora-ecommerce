// handler.go — HTTP handlers for OAuth redirect and callback.
package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

// Handler manages OAuth redirect and callback endpoints.
type Handler struct {
	service      Service
	providers    map[string]Provider
	baseURL      string
	cookieSecure bool
}

func NewHandler(service Service, providers map[string]Provider, baseURL string, cookieSecure bool) *Handler {
	return &Handler{
		service:      service,
		providers:    providers,
		baseURL:      baseURL,
		cookieSecure: cookieSecure,
	}
}

// Redirect sends the user to the OAuth provider's consent screen.
// GET /api/v1/auth/oauth/{provider}
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")

	provider, ok := h.providers[providerName]
	if !ok {
		response.Error(w, http.StatusBadRequest, "unsupported oauth provider")
		return
	}

	state, err := generateState()
	if err != nil {
		log.Printf("oauth state error [%s]: %v", providerName, err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Persist the state in a short-lived cookie so the callback can verify the
	// provider echoed back the same value — defeats OAuth login CSRF. SameSite
	// Lax lets the cookie ride the top-level GET redirect back from the provider.
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/api/v1/auth/oauth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})

	http.Redirect(w, r, provider.AuthURL(state), http.StatusTemporaryRedirect)
}

const oauthStateCookie = "oauth_state"

// Callback handles the OAuth provider's redirect back to our app.
// GET /api/v1/auth/oauth/{provider}/callback?code=...&state=...
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")

	provider, ok := h.providers[providerName]
	if !ok {
		response.Error(w, http.StatusBadRequest, "unsupported oauth provider")
		return
	}

	// Verify the state parameter against the cookie set in Redirect (CSRF).
	stateParam := r.URL.Query().Get("state")
	stateCookie, cookieErr := r.Cookie(oauthStateCookie)
	if cookieErr != nil || stateParam == "" || stateCookie.Value != stateParam {
		response.Error(w, http.StatusBadRequest, "invalid or missing oauth state")
		return
	}
	// Consume the state cookie so it can't be replayed.
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/api/v1/auth/oauth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		response.Error(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	userInfo, err := provider.Exchange(code)
	if err != nil {
		log.Printf("oauth exchange error [%s]: %v", providerName, err)
		response.Error(w, http.StatusUnauthorized, "oauth authentication failed")
		return
	}

	loginResp, err := h.service.OAuthLogin(r.Context(), userInfo)
	if err != nil {
		log.Printf("oauth login error [%s]: %v", providerName, err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// HttpOnly cookie carries the refresh token so it survives a full page
	// reload (e.g. the Stripe checkout redirect). Mirrors the email/password
	// login flow in internal/auth/handler.go.
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    loginResp.RefreshToken,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60,
	})

	// Redirect to the app. No tokens in the URL — the HttpOnly refresh cookie
	// above plus the SPA's mount-time refresh establish the session, so query/
	// fragment token leakage (browser history, referrer, server logs) is avoided.
	http.Redirect(w, r, h.baseURL+"/", http.StatusTemporaryRedirect)
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return hex.EncodeToString(b), nil
}
