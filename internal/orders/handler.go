package orders

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	mw "github.com/IbrahimmSenn/stora-ecommerce/internal/middleware"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/response"
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

	userID, guestID := mw.ResolveOwner(r)

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
	userID, guestID := mw.ResolveOwner(r)

	resp, err := h.service.GetByID(r.Context(), userID, guestID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// List handles GET /api/v1/orders?status=&from=&to=
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, guestID := mw.ResolveOwner(r)

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

// Prefill handles GET /api/v1/checkout/prefill. Returns the contact +
// shipping address from the logged-in user's most recent order. Responds
// 204 No Content for guests or for users with no prior orders — the
// frontend treats either as "nothing to prefill" without surfacing an error.
func (h *Handler) Prefill(w http.ResponseWriter, r *http.Request) {
	userID, _ := mw.ResolveOwner(r)
	if userID == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	prefill, err := h.service.GetLatestPrefill(r.Context(), *userID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	if prefill == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	response.JSON(w, http.StatusOK, prefill)
}

// Cancel handles POST /api/v1/orders/{id}/cancel
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}
	userID, guestID := mw.ResolveOwner(r)

	resp, err := h.service.Cancel(r.Context(), userID, guestID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// resolveOwner mirrors the cart handler. JWT user takes priority over guest cookie.

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
		response.Error(w, http.StatusUnprocessableEntity, response.FormatValidation(ve))
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
	case errors.Is(err, ErrInvalidStatus):
		response.Error(w, http.StatusUnprocessableEntity, "invalid shipping status (use processing, shipped, delivered, or cancelled)")
	case errors.Is(err, ErrNotRefundable):
		response.Error(w, http.StatusConflict, "this order cannot be refunded in its current status")
	case errors.Is(err, ErrAddressNotVerifiable):
		response.ErrorWithCode(w, http.StatusUnprocessableEntity, errorCodeFor(err),
			"We could not verify this shipping address. Please double-check the street, city, and postal code, or choose to use it anyway.")
	case errors.Is(err, ErrAddressVerificationUnavailable):
		response.ErrorWithCode(w, http.StatusServiceUnavailable, errorCodeFor(err),
			"Address verification is temporarily unavailable. You may retry, or place the order without verification.")
	default:
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

