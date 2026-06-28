// handler.go — HTTP handlers for login, logout, token refresh, password reset, and 2FA.
package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/ctxkey"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/passwordpolicy"
	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

// refreshCookieName is scoped to /api/v1/auth so the long-lived refresh
// token is only sent on the endpoints that need it (refresh, logout), not
// every API call.
const (
	refreshCookieName = "refresh_token"
	refreshCookiePath = "/api/v1/auth"
	refreshCookieTTL  = 7 * 24 * 60 * 60 // 7 days in seconds, matches token.go
)

type Handler struct {
	service      AuthService
	cookieSecure bool
}

func NewHandler(service AuthService, cookieSecure bool) *Handler {
	return &Handler{service: service, cookieSecure: cookieSecure}
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   refreshCookieTTL,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req LoginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
		case errors.Is(err, ErrInvalidCredentials):
			response.Error(w, http.StatusUnauthorized, "invalid email or password")
		case errors.Is(err, Err2FARequired):
			response.Error(w, http.StatusForbidden, "2fa verification required")
		case errors.Is(err, ErrInvalid2FACode):
			response.Error(w, http.StatusUnauthorized, "invalid 2fa code")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	h.setRefreshCookie(w, resp.RefreshToken)
	response.JSON(w, http.StatusOK, resp)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req RefreshRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	// Empty body is fine — the cookie may carry the token instead.
	if err := dec.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		if c, err := r.Cookie(refreshCookieName); err == nil {
			req.RefreshToken = c.Value
		}
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusUnauthorized, "no refresh token provided")
		return
	}

	resp, err := h.service.RefreshTokens(r.Context(), req)
	if err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
		case errors.Is(err, ErrInvalidToken),
			errors.Is(err, ErrTokenNotFound):
			h.clearRefreshCookie(w)
			response.Error(w, http.StatusUnauthorized, "invalid refresh token")
		case errors.Is(err, ErrTokenUsed):
			h.clearRefreshCookie(w)
			response.Error(w, http.StatusUnauthorized, "refresh token already used — all sessions revoked")
		case errors.Is(err, ErrTokenRevoked):
			h.clearRefreshCookie(w)
			response.Error(w, http.StatusUnauthorized, "refresh token has been revoked")
		case errors.Is(err, ErrExpiredToken):
			h.clearRefreshCookie(w)
			response.Error(w, http.StatusUnauthorized, "refresh token has expired")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	h.setRefreshCookie(w, resp.RefreshToken)
	response.JSON(w, http.StatusOK, resp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "missing authentication")
		return
	}

	if err := h.service.Logout(r.Context(), userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.clearRefreshCookie(w)
	response.JSON(w, http.StatusOK, AuthMessageResponse{Message: "logged out successfully"})
}

// --- Password reset ---

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req ForgotPasswordRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ForgotPassword(r.Context(), req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
			return
		}
		// Log the underlying reason; the client still sees the generic 500.
		log.Printf("auth: forgot-password failed: %v", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Always return success to prevent email enumeration.
	response.JSON(w, http.StatusOK, AuthMessageResponse{Message: "if an account with that email exists, a reset link has been sent"})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req ResetPasswordRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.ResetPassword(r.Context(), req); err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
		case errors.Is(err, passwordpolicy.ErrWeak):
			response.Error(w, http.StatusBadRequest, passwordpolicy.ErrWeak.Error())
		case errors.Is(err, ErrResetTokenNotFound):
			response.Error(w, http.StatusBadRequest, "invalid or expired reset token")
		case errors.Is(err, ErrResetTokenUsed):
			response.Error(w, http.StatusBadRequest, "reset token has already been used")
		case errors.Is(err, ErrResetTokenExpired):
			response.Error(w, http.StatusBadRequest, "reset token has expired")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.JSON(w, http.StatusOK, AuthMessageResponse{Message: "password has been reset successfully"})
}

// --- 2FA ---

func (h *Handler) Setup2FA(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	email, _ := r.Context().Value(ctxkey.Email).(string)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "missing authentication")
		return
	}

	resp, err := h.service.Setup2FA(r.Context(), userID, email)
	if err != nil {
		switch {
		case errors.Is(err, Err2FAAlreadyEnabled):
			response.Error(w, http.StatusConflict, "2fa is already enabled")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *Handler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "missing authentication")
		return
	}

	var req Verify2FARequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Enable2FA(r.Context(), userID, req); err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
		case errors.Is(err, Err2FAAlreadyEnabled):
			response.Error(w, http.StatusConflict, "2fa is already enabled")
		case errors.Is(err, Err2FANotEnabled):
			response.Error(w, http.StatusBadRequest, "2fa has not been set up — call setup first")
		case errors.Is(err, ErrInvalid2FACode):
			response.Error(w, http.StatusUnauthorized, "invalid 2fa code")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.JSON(w, http.StatusOK, AuthMessageResponse{Message: "2fa has been enabled"})
}

func (h *Handler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID, _ := r.Context().Value(ctxkey.UserID).(string)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "missing authentication")
		return
	}

	var req Verify2FARequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Disable2FA(r.Context(), userID, req); err != nil {
		var ve validator.ValidationErrors
		switch {
		case errors.As(err, &ve):
			response.Error(w, http.StatusBadRequest, formatValidationErrors(ve))
		case errors.Is(err, Err2FANotEnabled):
			response.Error(w, http.StatusBadRequest, "2fa is not enabled")
		case errors.Is(err, ErrInvalid2FACode):
			response.Error(w, http.StatusUnauthorized, "invalid 2fa code")
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.JSON(w, http.StatusOK, AuthMessageResponse{Message: "2fa has been disabled"})
}

func formatValidationErrors(ve validator.ValidationErrors) string {
	msg := "validation failed:"
	for _, fe := range ve {
		msg += " " + fe.Field() + " " + fe.Tag() + ";"
	}
	return msg
}
