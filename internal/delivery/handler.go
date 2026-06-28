// handler.go — HTTP handlers for delivery options (public list + admin CRUD).
package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// List (public) returns active options for the checkout shipping selector.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	opts, err := h.service.List(r.Context(), true)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, opts)
}

// AdminList returns all options including inactive ones.
func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	opts, err := h.service.List(r.Context(), false)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, opts)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req CreateRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	o, err := h.service.Create(r.Context(), req)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, o)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	id := chi.URLParam(r, "id")
	var req UpdateRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	o, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		h.writeMutationErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, o)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Error(w, http.StatusNotFound, "delivery option not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeMutationErr(w http.ResponseWriter, err error) {
	var ve validator.ValidationErrors
	switch {
	case errors.As(err, &ve):
		response.Error(w, http.StatusBadRequest, "label and a non-negative price are required")
	case errors.Is(err, ErrInvalidCode):
		response.Error(w, http.StatusBadRequest, ErrInvalidCode.Error())
	case errors.Is(err, ErrCodeExists):
		response.Error(w, http.StatusConflict, "a delivery option with that code already exists")
	case errors.Is(err, ErrNotFound):
		response.Error(w, http.StatusNotFound, "delivery option not found")
	default:
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
