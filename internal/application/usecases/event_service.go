package usecases

import (
	"context"
	"strings"

	"github.com/rachelJG/event-notification-service/internal/application/validation"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

// EventService implements all event-related use cases.
type EventService struct {
	Repo ports.EventRepository
}

// SubmitEvent validates and persists a new event.
func (s EventService) SubmitEvent(ctx context.Context, eventType string, payload []byte, idempotencyKey, clientID string) (string, error) {
	if err := validation.ValidateEvent(eventType, payload); err != nil {
		return "", apperror.InvalidArgument(err.Error(), err)
	}
	event, err := entities.NewEvent(eventType, idempotencyKey, payload)
	if err != nil {
		return "", apperror.InvalidArgument(err.Error(), err)
	}

	// Assign client_id from the authenticated API key (optional, may be empty for backward compatibility)
	event.ClientID = clientID

	id, err := s.Repo.Create(ctx, event)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetEvent retrieves an event by its ID.
func (s EventService) GetEvent(ctx context.Context, id string) (entities.Event, error) {
	if strings.TrimSpace(id) == "" {
		return entities.Event{}, apperror.InvalidArgument("id is required", nil)
	}
	return s.Repo.GetByID(ctx, id)
}
