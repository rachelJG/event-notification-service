package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	EventTypeUserRegistered         = "UserRegistered"
	EventTypePasswordResetRequested = "PasswordResetRequested"
	EventTypeOrderPaid              = "OrderPaid"
	EventTypeOrderShipped           = "OrderShipped"
)

type Event struct {
	ID             string
	Type           string
	IdempotencyKey string
	Payload        json.RawMessage
	OccurredAt     time.Time
	CreatedAt      time.Time
}

type UserRegisteredPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type PasswordResetRequestedPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type OrderPaidPayload struct {
	OrderID  string  `json:"order_id"`
	UserID   string  `json:"user_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type OrderShippedPayload struct {
	OrderID        string `json:"order_id"`
	UserID         string `json:"user_id"`
	Carrier        string `json:"carrier"`
	TrackingNumber string `json:"tracking_number"`
}

func ValidateEvent(eventType string, payload json.RawMessage) error {
	if strings.TrimSpace(eventType) == "" {
		return errors.New("event_type is required")
	}

	switch eventType {
	case EventTypeUserRegistered:
		var body UserRegisteredPayload
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
	case EventTypePasswordResetRequested:
		var body PasswordResetRequestedPayload
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
	case EventTypeOrderPaid:
		var body OrderPaidPayload
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
	case EventTypeOrderShipped:
		var body OrderShippedPayload
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
