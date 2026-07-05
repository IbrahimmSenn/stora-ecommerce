// handler.go — HTTP handlers for product search, detail, CRUD, and images.
package product

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/activity"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/imageproc"
	mw "gitea.kood.tech/ibrahimsen/i-love-shopping/internal/middleware"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

type Handler struct {
	service  Service
	activity activity.Logger
}

func NewHandler(service Service, logger activity.Logger) *Handler {
	if logger == nil {
		logger = activity.NoopLogger{}
	}
	return &Handler{service: service, activity: logger}
}

func resolveOwner(r *http.Request) (*uuid.UUID, *uuid.UUID) {
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

// Search handles GET /api/v1/products with query params for faceted search.
// Query params: q, category_id, brand_id, min_price, max_price, min_rating, sort, page, page_size
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := SearchParams{
		Query:  q.Get("q"),
		SortBy: q.Get("sort"),
	}

	if q.Get("on_sale") == "true" {
		params.OnSale = true
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
	if params.Query != "" {
		userID, guestID := resolveOwner(r)
		h.activity.LogSearch(r.Context(), userID, guestID, params.Query)
	}
	response.JSON(w, http.StatusOK, result)
}

// Suggest handles GET /api/v1/products/suggest?q=...
func (h *Handler) Suggest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	suggestions, err := h.service.Suggest(r.Context(), q)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, suggestions)
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
	userID, guestID := resolveOwner(r)
	h.activity.LogView(r.Context(), userID, guestID, &p.ID, p.CategoryID)
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
		if errors.Is(err, ErrInvalidSalePrice) {
			response.Error(w, http.StatusBadRequest, ErrInvalidSalePrice.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusCreated, p)
}

// BulkUpload handles POST /api/v1/admin/products/bulk. It accepts either a JSON
// array of products (Content-Type: application/json) or a CSV file
// (Content-Type: text/csv). Rows are created individually; per-row failures are
// returned alongside the success count rather than failing the whole batch.
func (h *Handler) BulkUpload(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ct := r.Header.Get("Content-Type")
	var reqs []CreateProductRequest
	var err error

	switch {
	case strings.HasPrefix(ct, "text/csv"), strings.HasPrefix(ct, "application/csv"):
		reqs, err = parseProductCSV(r.Body)
	default:
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		err = dec.Decode(&reqs)
	}
	if err != nil {
		response.Error(w, http.StatusBadRequest, "could not parse upload: "+err.Error())
		return
	}
	if len(reqs) == 0 {
		response.Error(w, http.StatusBadRequest, "no products found in upload")
		return
	}
	if len(reqs) > 1000 {
		response.Error(w, http.StatusRequestEntityTooLarge, "too many rows (max 1000 per upload)")
		return
	}

	result := h.service.BulkCreate(r.Context(), reqs)
	status := http.StatusCreated
	if result.Created == 0 {
		status = http.StatusUnprocessableEntity
	}
	response.JSON(w, status, result)
}

// parseProductCSV reads a header row then maps each subsequent row to a
// CreateProductRequest. Recognised columns: name, description, price,
// stock_quantity, category_id, brand_id, weight_g, dimensions_cm. Unknown
// columns are ignored; name and price are the only practically required fields.
func parseProductCSV(r io.Reader) ([]CreateProductRequest, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	col := make(map[string]int, len(header))
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}

	get := func(rec []string, name string) string {
		if i, ok := col[name]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}

	var out []CreateProductRequest
	line := 1
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", line, err)
		}

		req := CreateProductRequest{Name: get(rec, "name")}
		if v := get(rec, "description"); v != "" {
			req.Description = &v
		}
		if v := get(rec, "price"); v != "" {
			n, perr := strconv.ParseInt(v, 10, 64)
			if perr != nil {
				return nil, fmt.Errorf("row %d: price must be an integer (cents)", line)
			}
			req.Price = n
		}
		if v := get(rec, "stock_quantity"); v != "" {
			n, perr := strconv.Atoi(v)
			if perr != nil {
				return nil, fmt.Errorf("row %d: stock_quantity must be an integer", line)
			}
			req.StockQuantity = n
		}
		if v := get(rec, "category_id"); v != "" {
			req.CategoryID = &v
		}
		if v := get(rec, "brand_id"); v != "" {
			req.BrandID = &v
		}
		if v := get(rec, "weight_g"); v != "" {
			if n, perr := strconv.Atoi(v); perr == nil {
				req.WeightG = &n
			}
		}
		if v := get(rec, "dimensions_cm"); v != "" {
			if f, perr := strconv.ParseFloat(v, 64); perr == nil {
				req.DimensionsCm = &f
			}
		}
		out = append(out, req)
	}
	return out, nil
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
		case errors.Is(err, ErrInvalidSalePrice):
			response.Error(w, http.StatusBadRequest, ErrInvalidSalePrice.Error())
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

// UploadImage handles POST /api/v1/admin/products/{id}/images/upload.
// Multipart form: `image` (file), optional `is_primary` ("true"/"1"). The file
// is decoded and re-sized into thumbnail/card/full variants.
func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")

	// Cap the in-memory + total body size before touching the file.
	const maxUpload = 12 << 20 // 12 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		response.Error(w, http.StatusRequestEntityTooLarge, "image is too large (max 12 MB)")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("image")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "no image file provided (field 'image')")
		return
	}
	defer file.Close()
	_ = header

	isPrimary := r.FormValue("is_primary") == "true" || r.FormValue("is_primary") == "1"

	img, err := h.service.UploadImage(r.Context(), productID, file, isPrimary)
	if err != nil {
		switch {
		case errors.Is(err, imageproc.ErrNotAnImage):
			response.Error(w, http.StatusBadRequest, "that file is not a valid image (use JPEG or PNG)")
		case errors.Is(err, ErrUploadsDisabled):
			response.Error(w, http.StatusServiceUnavailable, "image uploads are not configured on this server")
		case errors.Is(err, ErrProductNotFound):
			response.Error(w, http.StatusNotFound, "product not found")
		default:
			response.Error(w, http.StatusInternalServerError, "could not process image")
		}
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
