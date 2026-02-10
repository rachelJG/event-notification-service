package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const EventTypeUserRegistered = "UserRegistered"

type Event struct {
	ID         string
	Type       string
	Payload    json.RawMessage
	OccurredAt time.Time
	CreatedAt  time.Time
}

type UserRegisteredPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
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
		return nil
	default:
		return errors.New("unsupported event_type")
	}
}
