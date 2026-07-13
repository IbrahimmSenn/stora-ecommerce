// handler.go — HTTP handler for user registration.
package user

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/IbrahimmSenn/stora-ecommerce/internal/ctxkey"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/passwordpolicy"
	"github.com/IbrahimmSenn/stora-ecommerce/internal/response"
)

type Handler struct {
	service UserService
}

func NewHandler(service UserService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req RegisterRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
		case errors.Is(err, passwordpolicy.ErrWeak):
			response.Error(w, http.StatusBadRequest, passwordpolicy.ErrWeak.Error())
		case errors.Is(err, ErrEmailExists):
			response.Error(w, http.StatusConflict, "email already taken")
		case errors.Is(err, ErrCaptchaInvalid):
			log.Printf("register: captcha rejected: %v", err)
			response.Error(w, http.StatusBadRequest, "captcha verification failed")
		default:
			log.Printf("register: internal error: %v", err)
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	me, err := h.service.GetMe(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.Error(w, http.StatusNotFound, "account not found")
			return
		}
		log.Printf("me: internal error: %v", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, me)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	userID, _ := r.Context().Value(ctxkey.UserID).(string)

	var req UpdateProfileRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	me, err := h.service.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrNameTooLong):
			response.Error(w, http.StatusBadRequest, ErrNameTooLong.Error())
		case errors.Is(err, ErrUserNotFound):
			response.Error(w, http.StatusNotFound, "account not found")
		default:
			log.Printf("update profile: internal error: %v", err)
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	response.JSON(w, http.StatusOK, me)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	userID, _ := r.Context().Value(ctxkey.UserID).(string)

	var req ChangePasswordRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ChangePassword(r.Context(), userID, req); err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
		case errors.Is(err, ErrWrongPassword):
			response.Error(w, http.StatusBadRequest, ErrWrongPassword.Error())
		case errors.Is(err, ErrNoPassword):
			response.Error(w, http.StatusBadRequest, ErrNoPassword.Error())
		case errors.Is(err, passwordpolicy.ErrWeak):
			response.Error(w, http.StatusBadRequest, passwordpolicy.ErrWeak.Error())
		case errors.Is(err, ErrUserNotFound):
			response.Error(w, http.StatusNotFound, "account not found")
		default:
			log.Printf("change password: internal error: %v", err)
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

func formatValidationErrors(ve validator.ValidationErrors) string {
	msg := "validation failed:"
	for _, fe := range ve {
		msg += " " + fe.Field() + " " + fe.Tag() + ";"
	}
	return msg
}
