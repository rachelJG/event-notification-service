package entities

import (
	"errors"
	"slices"
	"strings"
	"time"
)

// EventType is a value object representing a valid event type.
type EventType = string

const (
	EventTypeUserRegistered         EventType = "UserRegistered"
	EventTypePasswordResetRequested EventType = "PasswordResetRequested"
	EventTypeOrderPaid              EventType = "OrderPaid"
	EventTypeOrderShipped           EventType = "OrderShipped"
	EventTypeInvoiceIssued          EventType = "InvoiceIssued"
	EventTypeInvoiceSummary         EventType = "InvoiceSummary"
)

// ValidEventTypes returns the set of supported event types.
func ValidEventTypes() []EventType {
	return []EventType{
		EventTypeUserRegistered,
		EventTypePasswordResetRequested,
		EventTypeOrderPaid,
		EventTypeOrderShipped,
		EventTypeInvoiceIssued,
		EventTypeInvoiceSummary,
	}
}

// Payload types define the expected structure for each event type.
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

// InvoiceRecipient represents a single recipient within an InvoiceIssued event.
type InvoiceRecipient struct {
	Email    string  `json:"email"`
	Name     string  `json:"name"`
	UnitCode string  `json:"unit_code"`
	Amount   float64 `json:"amount"`
}

type InvoiceIssuedPayload struct {
	CondominiumID   string             `json:"condominium_id"`
	CondominiumName string             `json:"condominium_name"`
	InvoiceMonth    string             `json:"invoice_month"`
	DueDate         string             `json:"due_date"`
	Currency        string             `json:"currency"`
	Recipients      []InvoiceRecipient `json:"recipients"`
}

type InvoiceSummaryosePayload struct {
	CondominiumID   string  `json:"condominium_id"`
	CondominiumName string  `json:"condominium_name"`
	InvoiceMonth    string  `json:"invoice_month"`
	TotalUnits      int     `json:"total_units"`
	TotalAmount     float64 `json:"total_amount"`
	Currency        string  `json:"currency"`
	WhatsAppGroupID string  `json:"whatsapp_group_id"`
	Message         string  `json:"message"`
}

type Event struct {
	ID             string
	Type           string
	IdempotencyKey string
	Payload        []byte
	ClientID       string // Client identifier from API key metadata (optional for backward compatibility)
	OccurredAt     time.Time
	CreatedAt      time.Time
}

// NewEvent constructs an Event enforcing domain invariants.
func NewEvent(eventType, idempotencyKey string, payload []byte) (Event, error) {
	if strings.TrimSpace(eventType) == "" {
		return Event{}, errors.New("event_type is required")
	}
	if !slices.Contains(ValidEventTypes(), eventType) {
		return Event{}, errors.New("unsupported event_type")
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return Event{}, errors.New("idempotency_key is required")
	}
	return Event{
		Type:           eventType,
		IdempotencyKey: idempotencyKey,
		Payload:        payload,
		OccurredAt:     time.Now().UTC(),
	}, nil
}
