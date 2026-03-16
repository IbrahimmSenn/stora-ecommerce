package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

// Handler manages OAuth redirect and callback endpoints.
type Handler struct {
	service   Service
	providers map[string]Provider
}

func NewHandler(service Service, providers map[string]Provider) *Handler {
	return &Handler{service: service, providers: providers}
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

	state := generateState()
	url := provider.AuthURL(state)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Callback handles the OAuth provider's redirect back to our app.
// GET /api/v1/auth/oauth/{provider}/callback?code=...&state=...
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")

	provider, ok := h.providers[providerName]
	if !ok {
		response.Error(w, http.StatusBadRequest, "unsupported oauth provider")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		response.Error(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	userInfo, err := provider.Exchange(code)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "oauth authentication failed")
		return
	}

	loginResp, err := h.service.OAuthLogin(r.Context(), userInfo)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, loginResp)
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
