// handler.go — HTTP handlers for reviews, helpful voting, and moderation.
package review

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/ctxkey"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/response"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func viewerID(r *http.Request) *uuid.UUID {
	raw, ok := r.Context().Value(ctxkey.UserID).(string)
	if !ok || raw == "" {
		return nil
	}
	if uid, err := uuid.Parse(raw); err == nil {
		return &uid
	}
	return nil
}

// List handles GET /api/v1/products/{id}/reviews
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := ListParams{
		ProductID: chi.URLParam(r, "id"),
		Sort:      q.Get("sort"),
		ViewerID:  viewerID(r),
	}
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.Page = n
		}
	}
	if v := q.Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.PageSize = n
		}
	}

	result, err := h.service.List(r.Context(), params)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(w, http.StatusNotFound, "product not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// Create handles POST /api/v1/products/{id}/reviews (auth required).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	productID := chi.URLParam(r, "id")

	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req CreateReviewRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rv, err := h.service.Create(r.Context(), userID, productID, req)
	if err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, "rating must be between 1 and 5; comment max 2000 characters")
		case errors.Is(err, ErrNotPurchased):
			response.ErrorWithCode(w, http.StatusForbidden, "not_purchased", ErrNotPurchased.Error())
		case errors.Is(err, ErrAlreadyReviewed):
			response.ErrorWithCode(w, http.StatusConflict, "already_reviewed", ErrAlreadyReviewed.Error())
		case errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "product not found")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	response.JSON(w, http.StatusCreated, rv)
}

// Eligibility handles GET /api/v1/products/{id}/reviews/eligibility (auth required).
func (h *Handler) Eligibility(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	e, err := h.service.Eligibility(r.Context(), userID, productID)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(w, http.StatusNotFound, "product not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, e)
}

// Vote handles POST /api/v1/reviews/{id}/helpful (auth required).
func (h *Handler) Vote(w http.ResponseWriter, r *http.Request) {
	h.setVote(w, r, true)
}

// Unvote handles DELETE /api/v1/reviews/{id}/helpful (auth required).
func (h *Handler) Unvote(w http.ResponseWriter, r *http.Request) {
	h.setVote(w, r, false)
}

func (h *Handler) setVote(w http.ResponseWriter, r *http.Request, helpful bool) {
	reviewID := chi.URLParam(r, "id")
	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if err := h.service.Vote(r.Context(), reviewID, userID, helpful); err != nil {
		if errors.Is(err, ErrReviewNotFound) {
			response.Error(w, http.StatusNotFound, "review not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"voted": helpful})
}

// --- Admin moderation ---

// ListModeration handles GET /api/v1/admin/reviews?status=pending&page=
func (h *Handler) ListModeration(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	items, total, err := h.service.ListForModeration(r.Context(), status, page, pageSize)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"reviews": items, "total": total})
}

// SetStatus handles PATCH /api/v1/admin/reviews/{id} with body {"status": "..."}.
func (h *Handler) SetStatus(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	reviewID := chi.URLParam(r, "id")

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.SetStatus(r.Context(), reviewID, req.Status); err != nil {
		if errors.Is(err, ErrReviewNotFound) {
			response.Error(w, http.StatusNotFound, "review not found")
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

// Delete handles DELETE /api/v1/admin/reviews/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	reviewID := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), reviewID); err != nil {
		if errors.Is(err, ErrReviewNotFound) {
			response.Error(w, http.StatusNotFound, "review not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "review deleted"})
}
