package product

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

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

// Search handles GET /api/v1/products with query params for faceted search.
// Query params: q, category_id, brand_id, min_price, max_price, min_rating, sort, page, page_size
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := SearchParams{
		Query:  q.Get("q"),
		SortBy: q.Get("sort"),
	}

	if v := q.Get("category_id"); v != "" {
		params.CategoryID = &v
	}
	if v := q.Get("brand_id"); v != "" {
		params.BrandID = &v
	}
	if v := q.Get("min_price"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.MinPrice = &n
		}
	}
	if v := q.Get("max_price"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			params.MaxPrice = &n
		}
	}
	if v := q.Get("min_rating"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.MinRating = &n
		}
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

	result, err := h.service.Search(r.Context(), params)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// GetByID handles GET /api/v1/products/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	p, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(w, http.StatusNotFound, "product not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, p)
}

// Create handles POST /api/v1/admin/products
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req CreateProductRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.service.Create(r.Context(), req)
	if err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusCreated, p)
}

// Update handles PUT /api/v1/admin/products/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	id := chi.URLParam(r, "id")

	var req UpdateProductRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
		case errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "product not found")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	response.JSON(w, http.StatusOK, p)
}

// Delete handles DELETE /api/v1/admin/products/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(w, http.StatusNotFound, "product not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "product deleted"})
}

// AddImage handles POST /api/v1/admin/products/{id}/images
func (h *Handler) AddImage(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	productID := chi.URLParam(r, "id")

	var req AddImageRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	img, err := h.service.AddImage(r.Context(), productID, req)
	if err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			response.Error(w, http.StatusBadRequest, "url is required and must be valid")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusCreated, img)
}

// DeleteImage handles DELETE /api/v1/admin/products/{id}/images/{imageId}
func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	imageID := chi.URLParam(r, "imageId")

	if err := h.service.DeleteImage(r.Context(), productID, imageID); err != nil {
		if errors.Is(err, ErrImageNotFound) {
			response.Error(w, http.StatusNotFound, "image not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "image deleted"})
}

func formatValidationErrors(ve validator.ValidationErrors) string {
	msg := "validation failed:"
	for _, fe := range ve {
		msg += " " + fe.Field() + " " + fe.Tag() + ";"
	}
	return msg
}
