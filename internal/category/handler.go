// handler.go — HTTP handlers for category listing, slug lookup, and creation.
package category

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

// ListTree returns categories as a nested tree for browsing.
func (h *Handler) ListTree(w http.ResponseWriter, r *http.Request) {
	tree, err := h.service.ListTree(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if tree == nil {
		tree = []CategoryTree{}
	}
	response.JSON(w, http.StatusOK, tree)
}

func (h *Handler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	c, err := h.service.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			response.Error(w, http.StatusNotFound, "category not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req CreateCategoryRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.service.Create(r.Context(), req)
	if err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, "name and slug are required")
		case errors.Is(err, ErrCategoryExists):
			response.Error(w, http.StatusConflict, "category already exists")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	id := chi.URLParam(r, "id")

	var req UpdateCategoryRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, "name and slug are required")
		case errors.Is(err, ErrCategoryNotFound):
			response.Error(w, http.StatusNotFound, "category not found")
		case errors.Is(err, ErrCategoryExists):
			response.Error(w, http.StatusConflict, "another category already uses that slug")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.service.Delete(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrCategoryNotFound):
			response.Error(w, http.StatusNotFound, "category not found")
		case errors.Is(err, ErrCategoryInUse):
			response.Error(w, http.StatusConflict,
				"this category has products or subcategories — reassign or remove them before deleting")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
