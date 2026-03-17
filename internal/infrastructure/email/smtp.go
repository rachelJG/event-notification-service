// Package email provides email sending infrastructure adapters.
package email

import (
	"context"
	"crypto/tls"
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

// Send sends an email via SMTP with STARTTLS support. It respects context cancellation.
func (s *SMTPSender) Send(ctx context.Context, to, subject, body string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	msg := buildMessage(s.from, to, subject, body)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.dialAndSend(to, msg)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("smtp send canceled: %w", ctx.Err())
	case err := <-errCh:
		return err
	}
}

// dialAndSend connects to the SMTP server and attempts STARTTLS before authenticating and sending.
func (s *SMTPSender) dialAndSend(to string, msg []byte) error {
	addr := net.JoinHostPort(s.host, s.port)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	// Attempt STARTTLS; skip gracefully if not supported (e.g. dev SMTP servers).
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: s.host,
			MinVersion: tls.VersionTLS12,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	// Authenticate if credentials are provided.
	if s.user != "" && s.pass != "" {
		auth := smtp.PlainAuth("", s.user, s.pass, s.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp data close: %w", err)
	}

	return client.Quit()
}
