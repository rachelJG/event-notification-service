package ports

import "context"

// Output port: interface used by the application to send emails.
// The from parameter is the sender address provided by the client.
// If empty, the implementation should use its configured default.
type EmailSender interface {
	Send(ctx context.Context, from, to, subject, body string) error
}
