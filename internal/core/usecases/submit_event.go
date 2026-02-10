package usecases

import (
	"context"
	"time"

	"github.com/rachelJG/event-notification-service/internal/core/domain"
	"github.com/rachelJG/event-notification-service/internal/ports"
)

type SubmitEvent struct {
	Repo ports.EventRepository
}

func (uc SubmitEvent) Handle(ctx context.Context, eventType string, payload []byte) (string, error) {
	if err := domain.ValidateEvent(eventType, payload); err != nil {
		return "", err
	}

	event := domain.Event{
		Type:       eventType,
		Payload:    payload,
		OccurredAt: time.Now().UTC(),
	}

	return uc.Repo.Create(ctx, event)
}
