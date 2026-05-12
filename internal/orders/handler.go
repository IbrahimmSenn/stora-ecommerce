package orders

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	mw "gitea.kood.tech/ibrahimsen/i-love-shopping/internal/middleware"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Checkout handles POST /api/v1/checkout
func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req CheckoutRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, guestID := h.resolveOwner(r)

	resp, err := h.service.Checkout(r.Context(), userID, guestID, req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

// GetByID handles GET /api/v1/orders/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}
	userID, guestID := h.resolveOwner(r)

	resp, err := h.service.GetByID(r.Context(), userID, guestID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// List handles GET /api/v1/orders?status=&from=&to=
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, guestID := h.resolveOwner(r)

	q := r.URL.Query()
	status := q.Get("status")
	from, err := parseTimeParam(q.Get("from"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid 'from' (use RFC3339, e.g. 2026-01-15T00:00:00Z)")
		return
	}
	to, err := parseTimeParam(q.Get("to"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid 'to' (use RFC3339, e.g. 2026-01-15T00:00:00Z)")
		return
	}

	list, err := h.service.ListMine(r.Context(), userID, guestID, status, from, to)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, list)
}

// Cancel handles POST /api/v1/orders/{id}/cancel
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}
	userID, guestID := h.resolveOwner(r)

	resp, err := h.service.Cancel(r.Context(), userID, guestID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// resolveOwner mirrors the cart handler. JWT user takes priority over guest cookie.
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

func parseTimeParam(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var ve validator.ValidationErrors
	switch {
	case errors.As(err, &ve):
		response.Error(w, http.StatusUnprocessableEntity, formatValidationErrors(ve))
	case errors.Is(err, ErrCartEmpty):
		response.Error(w, http.StatusBadRequest, "your cart is empty")
	case errors.Is(err, ErrStockChanged):
		response.Error(w, http.StatusConflict, "stock or price changed while you were checking out — please review your cart")
	case errors.Is(err, ErrOrderNotFound):
		response.Error(w, http.StatusNotFound, "order not found")
	case errors.Is(err, ErrForbidden):
		response.Error(w, http.StatusForbidden, "you do not have access to this order")
	case errors.Is(err, ErrNotCancellable):
		response.Error(w, http.StatusConflict, "this order can no longer be cancelled")
	case errors.Is(err, ErrNoOwner):
		response.Error(w, http.StatusUnauthorized, "checkout requires a logged-in user or a guest session — please enable cookies")
	case errors.Is(err, ErrInvalidShipping):
		response.Error(w, http.StatusUnprocessableEntity, "invalid shipping method (use 'standard' or 'express')")
	case errors.Is(err, ErrRefundUnavailable):
		response.Error(w, http.StatusServiceUnavailable, "refunds are temporarily unavailable — please try again shortly")
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
