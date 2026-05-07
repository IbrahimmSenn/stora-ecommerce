package payments

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	mw "gitea.kood.tech/ibrahimsen/i-love-shopping/internal/middleware"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/orders"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// CreateIntent handles POST /api/v1/orders/{id}/payment-intent
func (h *Handler) CreateIntent(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}
	userID, guestID := h.resolveOwner(r)

	resp, err := h.service.CreateIntent(r.Context(), userID, guestID, id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// Webhook handles POST /api/v1/webhooks/stripe
//
// The Stripe signature is computed over the raw payload, so the handler must
// read the body before any JSON decode happens. We cap at 1 MiB — Stripe's
// own limit is 1 MB, anything larger is almost certainly an attack.
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "could not read webhook body")
		return
	}
	sig := r.Header.Get("Stripe-Signature")

	if err := h.service.HandleWebhook(r.Context(), payload, sig); err != nil {
		// Log the underlying reason so we can tell secret-mismatch from
		// timestamp-out-of-tolerance from missing-header without redeploying.
		log.Printf("stripe webhook rejected: %v (sig_header_present=%v, body_bytes=%d)", err, sig != "", len(payload))
		if errors.Is(err, ErrSignatureMismatch) {
			response.Error(w, http.StatusBadRequest, "invalid signature")
			return
		}
		response.Error(w, http.StatusInternalServerError, "webhook processing failed")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"received": "ok"})
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

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		response.Error(w, http.StatusForbidden, "you do not have access to this order")
	case errors.Is(err, ErrInvalidOrderStatus):
		response.Error(w, http.StatusConflict, "this order is not awaiting payment")
	case errors.Is(err, orders.ErrOrderNotFound):
		response.Error(w, http.StatusNotFound, "order not found")
	default:
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
