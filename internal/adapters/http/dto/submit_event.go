package dto

import "encoding/json"

type SubmitEventRequest struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

type SubmitEventResponse struct {
	ID string `json:"id"`
}
