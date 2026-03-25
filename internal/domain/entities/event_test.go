package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		eventType      string
		idempotencyKey string
		payload        []byte
		wantErr        string
	}{
		{
			name:           "valid UserRegistered event",
			eventType:      EventTypeUserRegistered,
			idempotencyKey: "idem-key-1",
			payload:        []byte(`{"user_id":"u1","email":"a@b.com","name":"Alice"}`),
		},
		{
			name:           "valid InvoiceSummary event",
			eventType:      EventTypeInvoiceSummary,
			idempotencyKey: "idem-key-2",
			payload:        []byte(`{}`),
		},
		{
			name:           "nil payload is accepted",
			eventType:      EventTypeOrderPaid,
			idempotencyKey: "idem-key-3",
			payload:        nil,
		},
		{
			name:           "empty payload is accepted",
			eventType:      EventTypeOrderShipped,
			idempotencyKey: "idem-key-4",
			payload:        []byte{},
		},
		{
			name:           "empty event type",
			eventType:      "",
			idempotencyKey: "idem-key",
			payload:        []byte(`{}`),
			wantErr:        "event_type is required",
		},
		{
			name:           "whitespace-only event type",
			eventType:      "   ",
			idempotencyKey: "idem-key",
			payload:        []byte(`{}`),
			wantErr:        "event_type is required",
		},
		{
			name:           "unsupported event type",
			eventType:      "UnknownType",
			idempotencyKey: "idem-key",
			payload:        []byte(`{}`),
			wantErr:        "unsupported event_type",
		},
		{
			name:           "case-sensitive event type",
			eventType:      "userregistered",
			idempotencyKey: "idem-key",
			payload:        []byte(`{}`),
			wantErr:        "unsupported event_type",
		},
		{
			name:           "empty idempotency key",
			eventType:      EventTypeUserRegistered,
			idempotencyKey: "",
			payload:        []byte(`{}`),
			wantErr:        "idempotency_key is required",
		},
		{
			name:           "whitespace-only idempotency key",
			eventType:      EventTypeUserRegistered,
			idempotencyKey: "   \t  ",
			payload:        []byte(`{}`),
			wantErr:        "idempotency_key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event, err := NewEvent(tt.eventType, tt.idempotencyKey, tt.payload)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error(), "error message mismatch")
				assert.Equal(t, Event{}, event, "should return zero-value Event on error")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.eventType, event.Type, "event type")
			assert.Equal(t, tt.idempotencyKey, event.IdempotencyKey, "idempotency key")
			assert.Equal(t, tt.payload, event.Payload, "payload")
			assert.False(t, event.OccurredAt.IsZero(), "OccurredAt should be set")
			assert.True(t, event.OccurredAt.Location().String() == "UTC", "OccurredAt should be UTC")
		})
	}
}

func TestNewEvent_AllValidTypesAccepted(t *testing.T) {
	t.Parallel()

	for _, et := range ValidEventTypes() {
		t.Run(et, func(t *testing.T) {
			t.Parallel()

			event, err := NewEvent(et, "idem-key", []byte(`{}`))
			require.NoError(t, err, "event type %s should be accepted", et)
			assert.Equal(t, et, event.Type)
		})
	}
}

func TestValidEventTypes(t *testing.T) {
	t.Parallel()

	types := ValidEventTypes()
	assert.Len(t, types, 6, "expected 6 event types")

	expected := []EventType{
		EventTypeUserRegistered,
		EventTypePasswordResetRequested,
		EventTypeOrderPaid,
		EventTypeOrderShipped,
		EventTypeInvoiceIssued,
		EventTypeInvoiceSummary,
	}
	assert.Equal(t, expected, types)
}
