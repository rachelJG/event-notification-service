package usecases

import (
	"context"
	"encoding/json"

	"github.com/rachelJG/event-notification-service/internal/application/validation"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
	"github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ProcessEvents struct {
	EventRepo        ports.EventRepository
	NotificationRepo ports.NotificationRepository
	BatchSize        int
}

func (uc ProcessEvents) Handle(ctx context.Context) (int, error) {
	ctx, span := otel.Tracer("usecases").Start(ctx, "ProcessEvents.Handle",
		trace.WithAttributes(attribute.Int("batch_size", uc.BatchSize)),
	)
	defer span.End()

	events, err := uc.EventRepo.ClaimPending(ctx, uc.BatchSize)
	if err != nil {
		span.SetStatus(codes.Error, "claim pending failed")
		span.RecordError(err)
		return 0, apperror.Internal("claim pending events", err)
	}

	span.SetAttributes(attribute.Int("events.claimed", len(events)))

	processed := 0
	for _, evt := range events {
		n, err := uc.processEvent(ctx, evt)
		if err != nil {
			_ = uc.EventRepo.SetStatus(ctx, evt.ID, "failed")
			continue
		}
		processed += n
	}

	span.SetAttributes(attribute.Int("events.processed", processed))
	return processed, nil
}

// processEvent creates notifications from the event's notification specs.
func (uc ProcessEvents) processEvent(ctx context.Context, evt entities.Event) (int, error) {
	ctx, span := otel.Tracer("usecases").Start(ctx, "ProcessEvents.processEvent",
		trace.WithAttributes(
			attribute.String("event.id", evt.ID),
			attribute.String("event.type", evt.Type),
		),
	)
	defer span.End()

	var specs []validation.NotificationSpec
	if err := json.Unmarshal(evt.NotificationsJSON, &specs); err != nil {
		span.SetStatus(codes.Error, "unmarshal failed")
		span.RecordError(err)
		return 0, apperror.Internal("unmarshal notification specs", err)
	}

	created := 0
	for _, spec := range specs {
		channel := entities.Channel(spec.Channel)
		for _, recipient := range spec.Recipients {
			n, err := entities.NewNotification(evt.ID, channel, spec.From, recipient, spec.Subject, spec.Body)
			if err != nil {
				span.SetStatus(codes.Error, "create notification failed")
				span.RecordError(err)
				return created, err
			}
			if _, err := uc.NotificationRepo.Create(ctx, n); err != nil {
				span.SetStatus(codes.Error, "persist notification failed")
				span.RecordError(err)
				return created, err
			}
			created++
		}
	}

	span.SetAttributes(attribute.Int("notifications.created", created))
	return created, nil
}
