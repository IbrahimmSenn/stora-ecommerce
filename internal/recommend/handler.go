package recommend

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/cart"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	mw "gitea.kood.tech/ibrahimsen/i-love-shopping/internal/middleware"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

const (
	defaultLimit = 4
	maxLimit     = 12
)

type Handler struct {
	service Service
	carts   cart.Service
}

func NewHandler(service Service, carts cart.Service) *Handler {
	return &Handler{service: service, carts: carts}
}

// Get handles GET /api/v1/recommendations?limit=N
//
// Resolves the shopper from the JWT context or the guest_session cookie,
// loads their current cart to use as short-term intent, and returns a
// scored rail of products.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, guestID := h.resolveOwner(r)

	limit := defaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > maxLimit {
				n = maxLimit
			}
			limit = n
		}
	}

	cartProductIDs := []string{}
	if userID != nil || guestID != nil {
		c, err := h.carts.GetCart(r.Context(), userID, guestID)
		if err == nil && c != nil {
			for _, it := range c.Items {
				cartProductIDs = append(cartProductIDs, it.ProductID.String())
			}
		}
	}

	items, err := h.service.Recommend(r.Context(), userID, guestID, cartProductIDs, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "could not load recommendations")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) resolveOwner(r *http.Request) (*uuid.UUID, *uuid.UUID) {
	if raw, ok := r.Context().Value(ctxkey.UserID).(string); ok && raw != "" {
		if uid, err := uuid.Parse(raw); err == nil {
			return &uid, nil
		}
	}
	if c, err := r.Cookie(mw.GuestSessionCookie); err == nil {
		if gid, err := uuid.Parse(c.Value); err == nil {
			return nil, &gid
		}
	}
	return nil, nil
}
