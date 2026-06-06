// Package email sends transactional email (project invitations) via SMTP, with
// a no-op fallback when SMTP is unconfigured so the rest of the system — and the
// dev/self-host experience (accept links are logged) — works without it.
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
)

// Mailer delivers transactional messages.
type Mailer interface {
	// Send delivers a plaintext message to one recipient.
	Send(ctx context.Context, to, subject, body string) error
	// Enabled reports whether a real transport is configured (vs. the no-op).
	Enabled() bool
}

// Noop is the fallback used when SMTP is unconfigured: Send does nothing.
type Noop struct{}

func (Noop) Send(context.Context, string, string, string) error { return nil }
func (Noop) Enabled() bool                                      { return false }

// Config configures the SMTP transport.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      string // "starttls" (default), "implicit", or "none"
}

// New returns an SMTP mailer when Host and From are set, otherwise a Noop.
func New(cfg Config) Mailer {
	if cfg.Host == "" || cfg.From == "" {
		return Noop{}
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	if cfg.TLS == "" {
		cfg.TLS = "starttls"
	}
	return &smtpMailer{cfg: cfg}
}

type smtpMailer struct{ cfg Config }

func (m *smtpMailer) Enabled() bool { return true }

func (m *smtpMailer) Send(_ context.Context, to, subject, body string) error {
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("email: invalid recipient %q: %w", to, err)
	}
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	msg := BuildMessage(m.cfg.From, to, subject, body)
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	if m.cfg.TLS == "implicit" {
		return m.sendImplicit(addr, auth, to, msg)
	}
	return m.sendStartTLS(addr, auth, to, msg, m.cfg.TLS == "none")
}

func (m *smtpMailer) sendStartTLS(addr string, auth smtp.Auth, to string, msg []byte, plain bool) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if !plain {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host}); err != nil {
				return err
			}
		}
	}
	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
	}
	return deliver(c, m.cfg.From, to, msg)
}

func (m *smtpMailer) sendImplicit(addr string, auth smtp.Auth, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.cfg.Host})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return err
	}
	defer c.Close()
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	return deliver(c, m.cfg.From, to, msg)
}

func deliver(c *smtp.Client, from, to string, msg []byte) error {
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write(msg); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// BuildMessage assembles an RFC 5322 plaintext message. Exported for testing.
// Header values are defended against CRLF injection: addresses are stripped of
// CR/LF and the subject is MIME-encoded (which folds any control bytes), so
// attacker-influenced inputs (invitee email, project name) cannot inject extra
// headers, recipients, or a forged body.
func BuildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + stripCRLF(from) + "\r\n")
	b.WriteString("To: " + stripCRLF(to) + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

// stripCRLF removes CR and LF so a header value cannot inject new header lines.
func stripCRLF(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}
