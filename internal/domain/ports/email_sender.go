package ports

import "context"

// Output port: interface used by the application to send emails.
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}
