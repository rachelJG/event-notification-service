package ports

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

// Output port: interface used by the application to communicate with the infrastructure (repository).
type EventRepository interface {
	Create(ctx context.Context, event entities.Event) (string, error)
}

// Input port: interface used by driving adapters (HTTP, gRPC, etc.) to communicate with the application.
type SubmitEventUseCase interface {
	Handle(ctx context.Context, eventType string, payload []byte, idempotencyKey string) (string, error)
}
