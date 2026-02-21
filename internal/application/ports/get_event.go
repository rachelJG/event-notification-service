package ports

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

// Input port for retrieving a single event by its ID.
type GetEventUseCase interface {
	Handle(ctx context.Context, id string) (entities.Event, error)
}
