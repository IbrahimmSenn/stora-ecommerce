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

// formatValidationErrors turns validator/v10's typed errors into a single
// shopper-readable sentence. The frontend renders this in an alert, so we
// want plain English, not field names and tag literals.
func formatValidationErrors(ve validator.ValidationErrors) string {
	if len(ve) == 0 {
		return "please check your details and try again"
	}
	parts := make([]string, 0, len(ve))
	for _, fe := range ve {
		parts = append(parts, friendlyField(fe.Field())+" "+friendlyTag(fe))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += "; " + p
	}
	return out
}

func friendlyField(name string) string {
	switch name {
	case "Email":
		return "Email"
	case "Phone":
		return "Phone"
	case "ShippingMethod":
		return "Shipping method"
	case "RecipientName":
		return "Recipient name"
	case "Line1":
		return "Address line 1"
	case "Line2":
		return "Address line 2"
	case "City":
		return "City"
	case "Region":
		return "State / region"
	case "PostalCode":
		return "Postal code"
	case "Country":
		return "Country"
	default:
		return name
	}
}

func friendlyTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "is too short (min " + fe.Param() + " characters)"
	case "max":
		return "is too long (max " + fe.Param() + " characters)"
	case "len":
		return "must be exactly " + fe.Param() + " characters"
	case "alpha":
		return "must contain letters only"
	case "iso3166_1_alpha2":
		return "must be a valid ISO 3166-1 alpha-2 country code (e.g. US, GB, EE)"
	case "oneof":
		return "must be one of: " + fe.Param()
	default:
		return "is invalid"
	}
}
