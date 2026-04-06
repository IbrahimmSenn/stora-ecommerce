// mailer.go — sends emails over SMTP. No-ops if SMTP is not configured.
package mailer

import (
	"fmt"
	"net/smtp"
)

// Mailer sends emails via SMTP.
type Mailer struct {
	host string
	port string
	user string
	pass string
	from string
}

// New creates a Mailer. If host is empty, Send will be a no-op (for dev environments).
func New(host, port, user, pass, from string) *Mailer {
	return &Mailer{host: host, port: port, user: user, pass: pass, from: from}
}

// Send sends a plain-text email.
func (m *Mailer) Send(to, subject, body string) error {
	if m.host == "" {
		// No SMTP configured — log and skip (dev mode).
		fmt.Printf("[mailer] SMTP not configured, skipping email to=%s subject=%s\n", to, subject)
		return nil
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, body)

	addr := m.host + ":" + m.port
	auth := smtp.PlainAuth("", m.user, m.pass, m.host)

	if err := smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}
