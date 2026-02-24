package usecases

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
)

type ProcessEvents struct {
	EventRepo        ports.EventRepository
	NotificationRepo ports.NotificationRepository
	Renderer         ports.EmailRenderer
	BatchSize        int
}

func (uc ProcessEvents) Handle(ctx context.Context) (int, error) {
	events, err := uc.EventRepo.ClaimPending(ctx, uc.BatchSize)
	if err != nil {
		return 0, fmt.Errorf("claim pending events: %w", err)
	}

	processed := 0
	for _, evt := range events {
		recipient, err := extractRecipient(evt)
		if err != nil {
			_ = uc.EventRepo.SetStatus(ctx, evt.ID, "failed")
			continue
		}

		subject, body, err := uc.Renderer.Render(evt)
		if err != nil {
			_ = uc.EventRepo.SetStatus(ctx, evt.ID, "failed")
			continue
		}

		n, err := entities.NewNotification(evt.ID, entities.ChannelEmail, recipient, subject, body)
		if err != nil {
			_ = uc.EventRepo.SetStatus(ctx, evt.ID, "failed")
			continue
		}

		if _, err := uc.NotificationRepo.Create(ctx, n); err != nil {
			_ = uc.EventRepo.SetStatus(ctx, evt.ID, "failed")
			continue
		}

		processed++
	}

	return processed, nil
}

// extractRecipient resolves the email address from the event payload based on event type.
func extractRecipient(evt entities.Event) (string, error) {
	switch evt.Type {
	case entities.EventTypeUserRegistered:
		var p entities.UserRegisteredPayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return "", err
		}
		return p.Email, nil
	case entities.EventTypePasswordResetRequested:
		var p entities.PasswordResetRequestedPayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return "", err
		}
		return p.Email, nil
	case entities.EventTypeOrderPaid:
		var p struct {
			Email string `json:"email"`
		}
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return "", err
		}
		return p.Email, nil
	case entities.EventTypeOrderShipped:
		var p struct {
			Email string `json:"email"`
		}
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return "", err
		}
		return p.Email, nil
	default:
		return "", fmt.Errorf("unsupported event type: %s", evt.Type)
	}
}
