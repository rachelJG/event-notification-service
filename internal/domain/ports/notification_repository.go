package ports

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

// Output port: interface used by the application to persist and query notifications.
type NotificationRepository interface {
	Create(ctx context.Context, n entities.Notification) (string, error)
	FindPending(ctx context.Context, limit int) ([]entities.Notification, error)
	UpdateStatus(ctx context.Context, id string, status entities.NotificationStatus, lastError string) error
}
