// handler.go — HTTP handlers for saved addresses. All routes require auth; the
// owner is taken from the JWT context so users only ever touch their own rows.
package address

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func userID(r *http.Request) string {
	s, _ := r.Context().Value(ctxkey.UserID).(string)
	return s
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), userID(r))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "could not load addresses")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{"addresses": list})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req AddressRequest
	if err := decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	addr, err := h.svc.Create(r.Context(), userID(r), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, addr)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req AddressRequest
	if err := decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	addr, err := h.svc.Update(r.Context(), userID(r), chi.URLParam(r, "id"), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, addr)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), userID(r), chi.URLParam(r, "id")); err != nil {
		writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "address removed"})
}

func (h *Handler) SetDefault(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SetDefault(r.Context(), userID(r), chi.URLParam(r, "id")); err != nil {
		writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "default updated"})
}

func decode(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	var ve validator.ValidationErrors
	switch {
	case errors.As(err, &ve):
		response.Error(w, http.StatusBadRequest, "please provide a recipient, street, city, region, postal code, and 2-letter country")
	case errors.Is(err, ErrNotFound):
		response.Error(w, http.StatusNotFound, "address not found")
	default:
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
