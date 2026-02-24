package email

import (
	"encoding/json"
	"fmt"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

// TemplateRenderer implements ports.EmailRenderer by rendering email subject
// and body based on the event type and payload.
type TemplateRenderer struct{}

// NewTemplateRenderer creates a new TemplateRenderer.
func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{}
}

// Render returns the email subject and body for the given event.
func (r *TemplateRenderer) Render(evt entities.Event) (subject, body string, err error) {
	switch evt.Type {
	case entities.EventTypeUserRegistered:
		var p entities.UserRegisteredPayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return "", "", fmt.Errorf("unmarshal UserRegistered payload: %w", err)
		}
		return "Welcome, " + p.Name,
			fmt.Sprintf("Hello %s, your account has been created successfully.", p.Name),
			nil

	case entities.EventTypePasswordResetRequested:
		var p entities.PasswordResetRequestedPayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return "", "", fmt.Errorf("unmarshal PasswordResetRequested payload: %w", err)
		}
		return "Reset your password",
			fmt.Sprintf("A password reset was requested for your account (user %s).", p.UserID),
			nil

	case entities.EventTypeOrderPaid:
		var p entities.OrderPaidPayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return "", "", fmt.Errorf("unmarshal OrderPaid payload: %w", err)
		}
		return fmt.Sprintf("Payment confirmation #%s", p.OrderID),
			fmt.Sprintf("Your payment of %.2f %s for order %s has been confirmed.", p.Amount, p.Currency, p.OrderID),
			nil

	case entities.EventTypeOrderShipped:
		var p entities.OrderShippedPayload
		if err := json.Unmarshal(evt.Payload, &p); err != nil {
			return "", "", fmt.Errorf("unmarshal OrderShipped payload: %w", err)
		}
		return fmt.Sprintf("Your order #%s has been shipped", p.OrderID),
			fmt.Sprintf("Your order %s has been shipped via %s. Tracking number: %s.", p.OrderID, p.Carrier, p.TrackingNumber),
			nil

	default:
		return "", "", fmt.Errorf("unsupported event type: %s", evt.Type)
	}
}
