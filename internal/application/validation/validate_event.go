package validation

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

func ValidateEvent(eventType string, payload []byte) error {
	switch eventType {
	case entities.EventTypeUserRegistered:
		var body entities.UserRegisteredPayload
		if err := json.Unmarshal(payload, &body); err != nil {
			return errors.New("invalid UserRegistered payload")
		}
		if strings.TrimSpace(body.UserID) == "" || strings.TrimSpace(body.Email) == "" || strings.TrimSpace(body.Name) == "" {
			return errors.New("UserRegistered requires user_id, email, and name")
		}
		if !strings.Contains(body.Email, "@") {
			return errors.New("UserRegistered requires a valid email")
		}
		return nil
	case entities.EventTypePasswordResetRequested:
		var body entities.PasswordResetRequestedPayload
		if err := json.Unmarshal(payload, &body); err != nil {
			return errors.New("invalid PasswordResetRequested payload")
		}
		if strings.TrimSpace(body.UserID) == "" || strings.TrimSpace(body.Email) == "" {
			return errors.New("PasswordResetRequested requires user_id and email")
		}
		if !strings.Contains(body.Email, "@") {
			return errors.New("PasswordResetRequested requires a valid email")
		}
		return nil
	case entities.EventTypeOrderPaid:
		var body entities.OrderPaidPayload
		if err := json.Unmarshal(payload, &body); err != nil {
			return errors.New("invalid OrderPaid payload")
		}
		if strings.TrimSpace(body.OrderID) == "" || strings.TrimSpace(body.UserID) == "" || strings.TrimSpace(body.Currency) == "" {
			return errors.New("OrderPaid requires order_id, user_id, and currency")
		}
		if body.Amount <= 0 {
			return errors.New("OrderPaid requires amount > 0")
		}
		return nil
	case entities.EventTypeOrderShipped:
		var body entities.OrderShippedPayload
		if err := json.Unmarshal(payload, &body); err != nil {
			return errors.New("invalid OrderShipped payload")
		}
		if strings.TrimSpace(body.OrderID) == "" || strings.TrimSpace(body.UserID) == "" || strings.TrimSpace(body.Carrier) == "" || strings.TrimSpace(body.TrackingNumber) == "" {
			return errors.New("OrderShipped requires order_id, user_id, carrier, and tracking_number")
		}
		return nil
	default:
		return errors.New("unsupported event_type")
	}
}
