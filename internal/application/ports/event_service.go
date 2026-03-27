package ports

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/application/validation"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

// Input port: event service exposed to driving adapters (HTTP, gRPC, etc.).
type EventService interface {
	SubmitEvent(ctx context.Context, eventType string, payload []byte, notifications []validation.NotificationSpec, idempotencyKey, clientID string) (string, error)
	GetEvent(ctx context.Context, id string) (entities.Event, error)
}
