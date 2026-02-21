package ports

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

// Output port: interface used by the application to communicate with the infrastructure (repository).
type EventRepository interface {
	Create(ctx context.Context, event entities.Event) (string, error)
	GetByID(ctx context.Context, id string) (entities.Event, error)
	ClaimPending(ctx context.Context, limit int) ([]entities.Event, error)
	SetStatus(ctx context.Context, id string, status string) error
}
