package ports

import (
	"context"
	"time"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

// NotificationUpdate holds the fields to update on a notification.
type NotificationUpdate struct {
	Status      entities.NotificationStatus
	Attempts    int
	LastError   string
	NextRetryAt time.Time
}

// Output port: interface used by the application to persist and query notifications.
type NotificationRepository interface {
	Create(ctx context.Context, n entities.Notification) (string, error)
	FindPending(ctx context.Context, limit int) ([]entities.Notification, error)
	UpdateStatus(ctx context.Context, id string, update NotificationUpdate) error
}
