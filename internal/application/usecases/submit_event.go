package usecases

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/application/validation"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
)

type SubmitEvent struct {
	Repo ports.EventRepository
}

func (uc SubmitEvent) Handle(ctx context.Context, eventType string, payload []byte, idempotencyKey string) (string, error) {

	if err := validation.ValidateEvent(eventType, payload); err != nil {
		return "", apperror.InvalidArgument(err.Error(), err)
	}
	event, err := entities.NewEvent(eventType, idempotencyKey, payload)
	if err != nil {
		return "", apperror.InvalidArgument(err.Error(), err)
	}

	id, err := uc.Repo.Create(ctx, event)
	if err != nil {
		return "", err
	}
	return id, nil
}
