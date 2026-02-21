package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/rachelJG/event-notification-service/internal/application/validation"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

type SubmitEvent struct {
	Repo ports.EventRepository
}

func (uc SubmitEvent) Handle(ctx context.Context, eventType string, payload []byte, idempotencyKey string) (string, error) {

	if strings.TrimSpace(eventType) == "" {
		return "", errors.New("event_type is required")
	}
	if !slices.Contains(entities.ValidEventTypes(), eventType) {
		return "", errors.New("unsupported event_type")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return "", errors.New("idempotency_key is required")
	}

	if err := validation.ValidateEvent(eventType, payload); err != nil {
		return "", apperror.InvalidArgument(err.Error(), err)
	}
	event, err := entities.NewEvent(eventType, idempotencyKey, payload)
	if err != nil {
		return "", apperror.InvalidArgument(err.Error(), err)
	}

	id, err := uc.Repo.Create(ctx, event)
	if err != nil {
		return "", apperror.Internal("failed to persist event", err)
	}
	return id, nil
}
