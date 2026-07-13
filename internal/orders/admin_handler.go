// admin_handler.go — HTTP handlers for the admin orders surface. Routes are
// guarded by staff RBAC; these handlers do not perform owner checks.
package orders

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/response"
)

// AdminList handles GET /api/v1/admin/orders?status=&from=&to=&page=&page_size=
func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, err := parseTimeParam(q.Get("from"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid 'from' (use RFC3339)")
		return
	}
	to, err := parseTimeParam(q.Get("to"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid 'to' (use RFC3339)")
		return
	}
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	list, err := h.service.AdminList(r.Context(), q.Get("status"), from, to, page, pageSize)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, list)
}

// AdminGet handles GET /api/v1/admin/orders/{id}
func (h *Handler) AdminGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}
	resp, err := h.service.AdminGet(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// AdminUpdateStatus handles PATCH /api/v1/admin/orders/{id}/status with body {"status": "..."}.
func (h *Handler) AdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.service.AdminUpdateStatus(r.Context(), id, req.Status)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// AdminRefund handles POST /api/v1/admin/orders/{id}/refund
func (h *Handler) AdminRefund(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}
	resp, err := h.service.AdminRefund(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}
