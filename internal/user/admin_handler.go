// admin_handler.go — HTTP handlers for admin user management. Guarded by
// admin-only RBAC at the route layer.
package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/response"
)

// AdminList handles GET /api/v1/admin/users?page=&page_size=
func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	list, err := h.service.AdminListUsers(r.Context(), page, pageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "could not load users")
		return
	}
	response.JSON(w, http.StatusOK, list)
}

// AdminSetRole handles PATCH /api/v1/admin/users/{id}/role with body {"role": "..."}.
func (h *Handler) AdminSetRole(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	userID := chi.URLParam(r, "id")

	var req SetRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.AdminSetRole(r.Context(), userID, req.Role); err != nil {
		switch {
		case errors.Is(err, ErrInvalidRole):
			response.Error(w, http.StatusBadRequest, "invalid role (use admin, support, sales, or customer)")
		case errors.Is(err, ErrUserNotFound):
			response.Error(w, http.StatusNotFound, "user not found")
		case errors.Is(err, ErrLastAdmin):
			response.Error(w, http.StatusConflict, "cannot remove the last admin account")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"role": req.Role})
}
