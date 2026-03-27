package dto

import (
	"encoding/json"
	"time"
)

type NotificationDTO struct {
	Channel    string   `json:"channel" binding:"required"`
	From       string   `json:"from,omitempty"`
	Subject    string   `json:"subject"`
	Body       string   `json:"body" binding:"required"`
	Recipients []string `json:"recipients" binding:"required"`
}

type SubmitEventRequest struct {
	EventType     string            `json:"event_type"`
	Payload       json.RawMessage   `json:"payload"`
	Notifications []NotificationDTO `json:"notifications" binding:"required"`
}

type SubmitEventResponse struct {
	ID string `json:"id"`
}

type GetEventResponse struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	ClientID   string          `json:"client_id,omitempty"`
	OccurredAt time.Time       `json:"occurred_at"`
	CreatedAt  time.Time       `json:"created_at"`
}
