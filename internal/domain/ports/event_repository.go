package ports

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

// EventRepository (output port) is the interface that defines the methods that must be implemented by a
// concrete event repository.
type EventRepository interface {
	//	Create a new event in the repository. It uses idempotency to avoid duplicates, allowing an
	//	explicit ID to be provided or generated automatically. If an event with the same idempotency
	//	key and type already exists, it updates the existing record. It returns the ID of the created
	//	or updated event, and an error if any occurred.
	Create(ctx context.Context, event entities.Event) (string, error)

	//GetByID retrieves an event by its ID from the repository. It returns the event and an error if any occurred.
	GetByID(ctx context.Context, id string) (entities.Event, error)

	//ClaimPending claims pending events from the repository. It updates the status of the specified number of
	//events to "processing". It returns the claimed events and an error if any occurred.
	ClaimPending(ctx context.Context, limit int) ([]entities.Event, error)

	//Update the status of an event in the repository. It returns an error if any occurred.
	SetStatus(ctx context.Context, id string, status string) error
}
