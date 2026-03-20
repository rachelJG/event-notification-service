package dto

import (
	"encoding/json"
	"time"
)

type SubmitEventRequest struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

type SubmitEventResponse struct {
	ID string `json:"id"`
}

type GetEventResponse struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	ClientID       string          `json:"client_id,omitempty"`
	OccurredAt     time.Time       `json:"occurred_at"`
	CreatedAt      time.Time       `json:"created_at"`
}
