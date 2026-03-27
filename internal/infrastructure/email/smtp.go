// Package email provides email sending infrastructure adapters.
package email

import (
	"context"
	"crypto/tls"
	"net"
	"net/smtp"
	"strings"

	"github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
// If from is empty, the configured default (s.from) is used.
func (s *SMTPSender) Send(ctx context.Context, from, to, subject, body string) error {
	ctx, span := otel.Tracer("smtp").Start(ctx, "SMTPSender.Send",
		trace.WithAttributes(
			attribute.String("smtp.to", to),
			attribute.String("smtp.host", s.host),
		),
	)
	defer span.End()

	select {
	case <-ctx.Done():
		span.SetStatus(codes.Error, "context canceled")
		return ctx.Err()
	default:
	}

	sender := s.from
	if from != "" {
		sender = from
	}
	msg := buildMessage(sender, to, subject, body)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.dialAndSend(sender, to, msg)
	}()

	select {
	case <-ctx.Done():
		span.SetStatus(codes.Error, "send canceled")
		return apperror.Canceled("smtp send canceled", ctx.Err())
	case err := <-errCh:
		if err != nil {
			span.SetStatus(codes.Error, "send failed")
			span.RecordError(err)
		}
		return err
	}
}

// dialAndSend connects to the SMTP server and attempts STARTTLS before authenticating and sending.
func (s *SMTPSender) dialAndSend(from, to string, msg []byte) error {
	addr := net.JoinHostPort(s.host, s.port)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return apperror.Unavailable("smtp dial failed", err)
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		_ = conn.Close()
		return apperror.Unavailable("smtp new client failed", err)
	}
	defer func() { _ = client.Close() }()

	// Attempt STARTTLS; skip gracefully if not supported (e.g. dev SMTP servers).
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: s.host,
			MinVersion: tls.VersionTLS12,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return apperror.Unavailable("smtp starttls failed", err)
		}
	}

	// Authenticate if credentials are provided.
	if s.user != "" && s.pass != "" {
		auth := smtp.PlainAuth("", s.user, s.pass, s.host)
		if err := client.Auth(auth); err != nil {
			return apperror.Unauthenticated("smtp auth failed", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return apperror.Unavailable("smtp mail command failed", err)
	}
	if err := client.Rcpt(to); err != nil {
		return apperror.Unavailable("smtp rcpt command failed", err)
	}

	w, err := client.Data()
	if err != nil {
		return apperror.Unavailable("smtp data command failed", err)
	}
	if _, err := w.Write(msg); err != nil {
		return apperror.Unavailable("smtp write failed", err)
	}
	if err := w.Close(); err != nil {
		return apperror.Unavailable("smtp data close failed", err)
	}

	return client.Quit()
}
