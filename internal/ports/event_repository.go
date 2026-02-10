package ports

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/core/domain"
)

type EventRepository interface {
	Create(ctx context.Context, event domain.Event) (string, error)
}
