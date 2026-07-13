package cart

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/ctxkey"
	mw "github.com/IbrahimmSenn/stora-ecommerce/internal/middleware"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/product"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/response"
)

type Handler struct {
	service      Service
	cookieSecure bool
}

func NewHandler(service Service, cookieSecure bool) *Handler {
	return &Handler{service: service, cookieSecure: cookieSecure}
}

// GetCart handles GET /api/v1/cart
func (h *Handler) GetCart(w http.ResponseWriter, r *http.Request) {
	userID, guestID := h.resolveOwner(r)

	cart, err := h.service.GetCart(r.Context(), userID, guestID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to load cart")
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

// AddItem handles POST /api/v1/cart/items
func (h *Handler) AddItem(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req AddItemRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, guestID := h.resolveOwner(r)

	cart, err := h.service.AddItem(r.Context(), userID, guestID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

// UpdateItem handles PUT /api/v1/cart/items/{productId}
func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	productID := chi.URLParam(r, "productId")

	var body struct {
		Quantity int `json:"quantity"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req := UpdateItemRequest{
		ProductID: productID,
		Quantity:  body.Quantity,
	}

	userID, guestID := h.resolveOwner(r)

	cart, err := h.service.UpdateItem(r.Context(), userID, guestID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

// RemoveItem handles DELETE /api/v1/cart/items/{productId}
func (h *Handler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")

	userID, guestID := h.resolveOwner(r)

	cart, err := h.service.RemoveItem(r.Context(), userID, guestID, productID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cart)
}

// ClearCart handles DELETE /api/v1/cart
func (h *Handler) ClearCart(w http.ResponseWriter, r *http.Request) {
	userID, guestID := h.resolveOwner(r)

	if err := h.service.ClearCart(r.Context(), userID, guestID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to clear cart")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "cart cleared"})
}

// GetMergeStatus handles GET /api/v1/cart/merge-status
// Requires the strict Auth middleware — it's only meaningful for a logged-in
// user. Reports whether a pending guest cart conflicts with the user cart.
func (h *Handler) GetMergeStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := authUserID(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	guestID := readGuestCookie(r)

	status, err := h.service.MergeStatus(r.Context(), userID, guestID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	// If nothing to resolve, or items were silently folded in, the cookie is stale.
	if guestID != nil && (status.AutoMerged || !status.Conflict) {
		h.clearGuestCookie(w)
	}
	response.JSON(w, http.StatusOK, status)
}

// PostMerge handles POST /api/v1/cart/merge
// Requires the strict Auth middleware and a guest_session_id cookie.
func (h *Handler) PostMerge(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID, ok := authUserID(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	guestID := readGuestCookie(r)
	if guestID == nil {
		response.Error(w, http.StatusBadRequest, "no guest session to merge")
		return
	}

	var req MergeRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cart, err := h.service.Merge(r.Context(), userID, *guestID, req.Strategy)
	if err != nil {
		h.handleError(w, err)
		return
	}
	h.clearGuestCookie(w)
	response.JSON(w, http.StatusOK, cart)
}

func authUserID(r *http.Request) (uuid.UUID, bool) {
	raw, ok := r.Context().Value(ctxkey.UserID).(string)
	if !ok || raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func readGuestCookie(r *http.Request) *uuid.UUID {
	c, err := r.Cookie(mw.GuestSessionCookie)
	if err != nil {
		return nil
	}
	id, err := uuid.Parse(c.Value)
	if err != nil {
		return nil
	}
	return &id
}

func (h *Handler) clearGuestCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     mw.GuestSessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// resolveOwner extracts the user ID from the JWT context or the guest session
// ID from the cookie. Authenticated users take priority over guest sessions.
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

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var ve validator.ValidationErrors
	switch {
	case errors.As(err, &ve):
		response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
	case errors.Is(err, ErrOutOfStock):
		response.Error(w, http.StatusConflict, "not enough stock available")
	case errors.Is(err, ErrItemNotFound):
		response.Error(w, http.StatusNotFound, "item not found in cart")
	case errors.Is(err, ErrNoOwner):
		response.Error(w, http.StatusUnauthorized, "no user session found — please log in or enable cookies")
	case errors.Is(err, ErrInvalidStrategy):
		response.Error(w, http.StatusBadRequest, "strategy must be \"guest\" or \"user\"")
	case errors.Is(err, ErrNoGuestCookie):
		response.Error(w, http.StatusBadRequest, "no guest session to merge")
	case errors.Is(err, product.ErrProductNotFound):
		response.Error(w, http.StatusNotFound, "product not found")
	default:
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func formatValidationErrors(ve validator.ValidationErrors) string {
	msg := "validation failed:"
	for _, fe := range ve {
		msg += " " + fe.Field() + " " + fe.Tag() + ";"
	}
	return msg
}
