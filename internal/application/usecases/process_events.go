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

		subject, body := renderEmail(evt)

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

// renderEmail returns a subject and body for the notification based on event type and payload.
func renderEmail(evt entities.Event) (subject, body string) {
	switch evt.Type {
	case entities.EventTypeUserRegistered:
		var p entities.UserRegisteredPayload
		_ = json.Unmarshal(evt.Payload, &p)
		return "Welcome, " + p.Name, fmt.Sprintf("Hello %s, your account has been created successfully.", p.Name)
	case entities.EventTypePasswordResetRequested:
		var p entities.PasswordResetRequestedPayload
		_ = json.Unmarshal(evt.Payload, &p)
		return "Reset your password", fmt.Sprintf("A password reset was requested for your account (user %s).", p.UserID)
	case entities.EventTypeOrderPaid:
		var p entities.OrderPaidPayload
		_ = json.Unmarshal(evt.Payload, &p)
		return fmt.Sprintf("Payment confirmation #%s", p.OrderID), fmt.Sprintf("Your payment of %.2f %s for order %s has been confirmed.", p.Amount, p.Currency, p.OrderID)
	case entities.EventTypeOrderShipped:
		var p entities.OrderShippedPayload
		_ = json.Unmarshal(evt.Payload, &p)
		return fmt.Sprintf("Your order #%s has been shipped", p.OrderID), fmt.Sprintf("Your order %s has been shipped via %s. Tracking number: %s.", p.OrderID, p.Carrier, p.TrackingNumber)
	default:
		return "Notification", "You have a new notification."
	}
}
