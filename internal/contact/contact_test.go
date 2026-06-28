package contact

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMailer struct {
	to, subject, body string
	called            bool
	err               error
}

func (m *mockMailer) Send(to, subject, body string) error {
	m.called = true
	m.to, m.subject, m.body = to, subject, body
	return m.err
}

func validReq() Request {
	return Request{Name: "Jane", Email: "jane@example.com", Subject: "Order", Message: "Where is it?"}
}

func TestSubmit_SendsToSupportInbox(t *testing.T) {
	m := &mockMailer{}
	svc := NewService(m, "support@shop.com")

	require.NoError(t, svc.Submit(validReq()))
	assert.True(t, m.called)
	assert.Equal(t, "support@shop.com", m.to)
	assert.Contains(t, m.subject, "Order")
	assert.Contains(t, m.body, "jane@example.com")
}

func TestSubmit_RejectsInvalid(t *testing.T) {
	m := &mockMailer{}
	svc := NewService(m, "support@shop.com")

	req := validReq()
	req.Email = "not-an-email"
	assert.Error(t, svc.Submit(req))
	assert.False(t, m.called, "must not email on invalid input")
}

func TestHandler_StatusCodes(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		mailErr  error
		wantCode int
	}{
		{"valid", `{"name":"J","email":"j@x.com","subject":"S","message":"M"}`, nil, http.StatusOK},
		{"invalid email", `{"name":"J","email":"bad","subject":"S","message":"M"}`, nil, http.StatusBadRequest},
		{"bad json", `{not json`, nil, http.StatusBadRequest},
		{"mailer down", `{"name":"J","email":"j@x.com","subject":"S","message":"M"}`, errors.New("smtp fail"), http.StatusBadGateway},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := NewHandler(NewService(&mockMailer{err: c.mailErr}, "support@shop.com"))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/contact", strings.NewReader(c.body))
			rr := httptest.NewRecorder()
			h.Submit(rr, req)
			assert.Equal(t, c.wantCode, rr.Code)
		})
	}
}
