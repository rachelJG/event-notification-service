package entities

import (
	"testing"
)

func TestNewEventSuccess(t *testing.T) {
	event, err := NewEvent(EventTypeUserRegistered, "idem-key", []byte(`{}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if event.Type != EventTypeUserRegistered {
		t.Fatalf("expected type %s, got %s", EventTypeUserRegistered, event.Type)
	}
	if event.IdempotencyKey != "idem-key" {
		t.Fatalf("expected idempotency key to be set")
	}
	if event.OccurredAt.IsZero() {
		t.Fatalf("expected OccurredAt to be set")
	}
}

func TestNewEventEmptyType(t *testing.T) {
	_, err := NewEvent("", "idem-key", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for empty event_type")
	}
}

func TestNewEventUnsupportedType(t *testing.T) {
	_, err := NewEvent("UnknownType", "idem-key", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for unsupported event_type")
	}
}

func TestNewEventEmptyIdempotencyKey(t *testing.T) {
	_, err := NewEvent(EventTypeUserRegistered, "", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for empty idempotency_key")
	}
}

func TestNewEventAllTypesAccepted(t *testing.T) {
	for _, et := range ValidEventTypes() {
		_, err := NewEvent(et, "idem-key", []byte(`{}`))
		if err != nil {
			t.Fatalf("expected %s to be accepted, got error: %v", et, err)
		}
	}
}
