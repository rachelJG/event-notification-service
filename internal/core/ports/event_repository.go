package ports

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/core/domain"
)

// Output port: interface used by the application to communicate with the infrastructure (repository).
type EventRepository interface {
	Create(ctx context.Context, event domain.Event) (string, error)
}

// Input port: interfaz que los adapters driving (HTTP, gRPC, etc.) usan para hablar con la aplicacion.
type SubmitEventUseCase interface {
	Handle(ctx context.Context, eventType string, payload []byte, idempotencyKey string) (string, error)
}
