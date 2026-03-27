package usecases

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/rachelJG/event-notification-service/internal/application/validation"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// EventService implements all event-related use cases.
type EventService struct {
	Repo ports.EventRepository
}

// SubmitEvent validates and persists a new event.
func (s EventService) SubmitEvent(ctx context.Context, eventType string, payload []byte, notifications []validation.NotificationSpec, idempotencyKey, clientID string) (string, error) {
	ctx, span := otel.Tracer("usecases").Start(ctx, "EventService.SubmitEvent",
		trace.WithAttributes(
			attribute.String("event.type", eventType),
			attribute.String("event.client_id", clientID),
			attribute.Int("event.notifications_count", len(notifications)),
		),
	)
	defer span.End()

	if err := validation.ValidateEvent(eventType, payload, notifications); err != nil {
		span.SetStatus(codes.Error, "validation failed")
		span.RecordError(err)
		return "", apperror.InvalidArgument(err.Error(), err)
	}

	notificationsJSON, err := json.Marshal(notifications)
	if err != nil {
		span.SetStatus(codes.Error, "marshal failed")
		span.RecordError(err)
		return "", apperror.Internal("marshal notifications", err)
	}

	event, err := entities.NewEvent(eventType, idempotencyKey, payload, notificationsJSON)
	if err != nil {
		span.SetStatus(codes.Error, "domain validation failed")
		span.RecordError(err)
		return "", apperror.InvalidArgument(err.Error(), err)
	}

	// Assign client_id from the authenticated API key (optional, may be empty for backward compatibility)
	event.ClientID = clientID

	id, err := s.Repo.Create(ctx, event)
	if err != nil {
		span.SetStatus(codes.Error, "create failed")
		span.RecordError(err)
		return "", err
	}

	span.SetAttributes(attribute.String("event.id", id))
	return id, nil
}

// GetEvent retrieves an event by its ID.
func (s EventService) GetEvent(ctx context.Context, id string) (entities.Event, error) {
	if strings.TrimSpace(id) == "" {
		return entities.Event{}, apperror.InvalidArgument("id is required", nil)
	}
	return s.Repo.GetByID(ctx, id)
}
