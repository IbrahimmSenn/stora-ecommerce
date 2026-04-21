package cart

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/product"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

const guestSessionCookie = "guest_session_id"

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
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

// resolveOwner extracts the user ID from the JWT context or the guest session
// ID from the cookie. Authenticated users take priority over guest sessions.
func (h *Handler) resolveOwner(r *http.Request) (*uuid.UUID, *uuid.UUID) {
	if raw, ok := r.Context().Value(ctxkey.UserID).(string); ok && raw != "" {
		if uid, err := uuid.Parse(raw); err == nil {
			return &uid, nil
		}
	}

	if c, err := r.Cookie(guestSessionCookie); err == nil {
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
