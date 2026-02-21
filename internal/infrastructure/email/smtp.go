// Package email provides email sending infrastructure adapters.
package email

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTPSender implements ports.EmailSender using net/smtp.
type SMTPSender struct {
	host string
	port string
	user string
	pass string
	from string
}

// NewSMTPSender creates a new SMTP email sender.
func NewSMTPSender(host, port, user, password, from string) *SMTPSender {
	return &SMTPSender{
		host: host,
		port: port,
		user: user,
		pass: password,
		from: from,
	}
}

// buildMessage constructs an RFC 2822 email message with headers.
func buildMessage(from, to, subject, body string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}

// Send sends an email via SMTP. It respects context cancellation.
func (s *SMTPSender) Send(ctx context.Context, to, subject, body string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	msg := buildMessage(s.from, to, subject, body)
	addr := net.JoinHostPort(s.host, s.port)

	var auth smtp.Auth
	if s.user != "" && s.pass != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, s.from, []string{to}, msg)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("smtp send canceled: %w", ctx.Err())
	case err := <-errCh:
		return err
	}
}
