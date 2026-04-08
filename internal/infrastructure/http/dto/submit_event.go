package dto

import (
	"encoding/json"
	"time"
)

// NotificationDTO represents a notification to be sent via a specific channel
type NotificationDTO struct {
	Channel    string   `json:"channel" binding:"required" example:"email"`               // Notification channel (email, whatsapp)
	From       string   `json:"from,omitempty" example:"noreply@example.com"`             // Sender address (required for email)
	Subject    string   `json:"subject" example:"Invoice #12345"`                         // Subject line (used for email)
	Body       string   `json:"body" binding:"required" example:"Your invoice is ready"`  // Message body
	Recipients []string `json:"recipients" binding:"required" example:"user@example.com"` // List of recipient addresses
}

// SubmitEventRequest represents the request body for submitting an event
type SubmitEventRequest struct {
	EventType     string            `json:"event_type" binding:"required" example:"invoice.issued"` // Type of event (e.g., invoice.issued, payment.received)
	Payload       json.RawMessage   `json:"payload" swaggertype:"object,string"`                    // Event-specific data (varies by event_type)
	Notifications []NotificationDTO `json:"notifications" binding:"required"`                       // List of notifications to send for this event
}

// SubmitEventResponse represents the response after successfully submitting an event
type SubmitEventResponse struct {
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"` // Unique event identifier (UUID)
}

// GetEventResponse represents the response when retrieving an event
type GetEventResponse struct {
	ID         string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"` // Unique event identifier (UUID)
	Type       string          `json:"type" example:"invoice.issued"`                     // Type of event
	Payload    json.RawMessage `json:"payload" swaggertype:"object,string"`               // Event-specific data
	ClientID   string          `json:"client_id,omitempty" example:"acme-corp"`           // Client identifier from API key metadata
	OccurredAt time.Time       `json:"occurred_at" example:"2024-01-15T10:30:00Z"`        // When the event occurred
	CreatedAt  time.Time       `json:"created_at" example:"2024-01-15T10:30:01Z"`         // When the event was created in the system
}
