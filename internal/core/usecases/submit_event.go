package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/rachelJG/event-notification-service/internal/core/apperror"
	"github.com/rachelJG/event-notification-service/internal/core/domain"
	"github.com/rachelJG/event-notification-service/internal/core/ports"
)

type SubmitEvent struct {
	Repo ports.EventRepository
}

func (uc SubmitEvent) Handle(ctx context.Context, eventType string, payload []byte, idempotencyKey string) (string, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return "", apperror.InvalidArgument("idempotency_key is required", nil)
	}
	if err := domain.ValidateEvent(eventType, payload); err != nil {
		return "", apperror.InvalidArgument("invalid event", err)
	}

	event := domain.Event{
		Type:           eventType,
		IdempotencyKey: idempotencyKey,
		Payload:        payload,
		OccurredAt:     time.Now().UTC(),
	}

	id, err := uc.Repo.Create(ctx, event)
	if err != nil {
		return "", apperror.Internal("failed to persist event", err)
	}
	return id, nil
}
