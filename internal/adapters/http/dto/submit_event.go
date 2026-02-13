package dto

import "encoding/json"

type SubmitEventRequest struct {
	EventType string          `json:"event_type" binding:"required"`
	Payload   json.RawMessage `json:"payload" binding:"required"`
}

type SubmitEventResponse struct {
	ID string `json:"id"`
}
