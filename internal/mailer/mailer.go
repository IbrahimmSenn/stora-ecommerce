// Package mailer sends transactional email over SMTP. It speaks the protocol
// directly via net/smtp.Client so AUTH and STARTTLS are negotiated against
// what the server actually advertises — Mailhog (no AUTH, no TLS), Mailtrap
// (STARTTLS + AUTH on 587), and most providers (implicit TLS on 465) all
// work with the same code path and the same env shape.
package mailer

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
)

// Mailer sends emails via SMTP. Construct with New; send with Send.
type Mailer struct {
	host string
	port string
	user string
	pass string
	from string
}

// New creates a Mailer. If host is empty, Send is a no-op (dev fallback for
// environments without any SMTP at all).
func New(host, port, user, pass, from string) *Mailer {
	return &Mailer{host: host, port: port, user: user, pass: pass, from: from}
}

// Send delivers a single HTML email. The flow is:
//
//  1. Dial (plain TCP, or TLS for port 465).
//  2. EHLO with a hostname derived from the From address.
//  3. STARTTLS if the server advertises it and we aren't already on TLS.
//  4. AUTH PLAIN if we have a username AND the server advertises AUTH.
//     If we have a username but the server doesn't advertise AUTH (e.g.
//     Mailhog with a stale Mailtrap user lying around in .env), log a
//     warning and proceed unauthenticated — Mailhog accepts the message,
//     which is the only sane outcome in dev. A server that actually
//     required auth would have advertised it.
//  5. MAIL FROM / RCPT TO / DATA / QUIT.
func (m *Mailer) Send(to, subject, body string) error {
	if m.host == "" {
		log.Printf("[mailer] SMTP not configured, skipping email to=%s subject=%q", to, subject)
		return nil
	}

	addr := net.JoinHostPort(m.host, m.port)
	client, err := m.dial(addr)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	if err := client.Hello(ehloHost(m.from)); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}

	if m.port != "465" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	if m.user != "" {
		switch {
		case !authAdvertised(client):
			log.Printf("[mailer] SMTP_USER set but %s does not advertise AUTH; sending unauthenticated", m.host)
		case !connectionEncrypted(client):
			// Refuse to leak credentials over a plain connection. Dev servers
			// like Mailhog advertise AUTH but don't require it, so we proceed
			// without authenticating; a server that *required* auth would
			// reject MAIL FROM and we'd surface that real failure instead.
			log.Printf("[mailer] %s advertised AUTH on an unencrypted connection; sending unauthenticated (set up STARTTLS or use port 465 to enable auth)", m.host)
		default:
			auth := smtp.PlainAuth("", m.user, m.pass, m.host)
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, body,
	)
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}

// dial opens an SMTP client. Port 465 is implicit TLS by convention; anything
// else dials plain and lets STARTTLS upgrade later.
func (m *Mailer) dial(addr string) (*smtp.Client, error) {
	if m.port == "465" {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.host})
		if err != nil {
			return nil, fmt.Errorf("smtp tls dial: %w", err)
		}
		client, err := smtp.NewClient(conn, m.host)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("smtp new client: %w", err)
		}
		return client, nil
	}
	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("smtp dial: %w", err)
	}
	return client, nil
}

// ehloHost picks a plausible EHLO hostname. RFC 5321 wants a valid FQDN; the
// domain of the From address is the closest thing we always have.
func ehloHost(from string) string {
	if i := strings.LastIndex(from, "@"); i > 0 && i+1 < len(from) {
		return from[i+1:]
	}
	return "localhost"
}

// authAdvertised reports whether the SMTP server's EHLO response listed AUTH.
func authAdvertised(c *smtp.Client) bool {
	ok, _ := c.Extension("AUTH")
	return ok
}

// connectionEncrypted reports whether the SMTP client is currently talking
// over TLS — either implicit (dialed via tls.Dial on port 465) or upgraded
// via STARTTLS. PLAIN AUTH is only safe over an encrypted connection.
func connectionEncrypted(c *smtp.Client) bool {
	_, ok := c.TLSConnectionState()
	return ok
}
