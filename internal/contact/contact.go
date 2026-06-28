// Package contact handles the public contact/support form: validate the
// submission and forward it to the support inbox via the mailer.
package contact

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

// Mailer is the slice of the mailer the contact service needs.
type Mailer interface {
	Send(to, subject, body string) error
}

type Request struct {
	Name    string `json:"name" validate:"required,min=1,max=120"`
	Email   string `json:"email" validate:"required,email"`
	Subject string `json:"subject" validate:"required,min=1,max=160"`
	Message string `json:"message" validate:"required,min=1,max=4000"`
}

type Service struct {
	mail     Mailer
	to       string
	validate *validator.Validate
}

func NewService(mail Mailer, to string) *Service {
	return &Service{mail: mail, to: to, validate: validator.New()}
}

func (s *Service) Submit(req Request) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Subject = strings.TrimSpace(req.Subject)
	req.Message = strings.TrimSpace(req.Message)
	if err := s.validate.Struct(req); err != nil {
		return err
	}

	body := fmt.Sprintf("From: %s <%s>\n\nSubject: %s\n\n%s",
		req.Name, req.Email, req.Subject, req.Message)
	if err := s.mail.Send(s.to, "Contact form: "+req.Subject, body); err != nil {
		return fmt.Errorf("send contact email: %w", err)
	}
	return nil
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Submit handles POST /api/v1/contact.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req Request
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.Submit(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			response.Error(w, http.StatusBadRequest, "please fill in your name, a valid email, a subject, and a message")
			return
		}
		response.Error(w, http.StatusBadGateway, "we couldn't send your message right now — please try again shortly")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "thanks — we'll be in touch"})
}
