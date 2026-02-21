package ports

import "context"

// Input port for claiming pending events and creating notifications.
type ProcessEventsUseCase interface {
	Handle(ctx context.Context) (int, error)
}
