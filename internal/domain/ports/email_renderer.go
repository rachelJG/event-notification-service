package ports

import "github.com/rachelJG/event-notification-service/internal/domain/entities"

// EmailRenderer resolves the subject and body for a notification email
// based on the event type and payload.
type EmailRenderer interface {
	Render(evt entities.Event) (subject, body string, err error)
}
