package entities

import (
	"testing"
)

func TestNewNotification_Success(t *testing.T) {
	n, err := NewNotification("event-123", ChannelEmail, "user@example.com", "Welcome", "Hello!")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n.EventID != "event-123" {
		t.Errorf("expected event_id event-123, got %s", n.EventID)
	}
	if n.Channel != ChannelEmail {
		t.Errorf("expected channel email, got %s", n.Channel)
	}
	if n.Recipient != "user@example.com" {
		t.Errorf("expected recipient user@example.com, got %s", n.Recipient)
	}
	if n.Subject != "Welcome" {
		t.Errorf("expected subject Welcome, got %s", n.Subject)
	}
	if n.Body != "Hello!" {
		t.Errorf("expected body Hello!, got %s", n.Body)
	}
	if n.Status != NotificationStatusPending {
		t.Errorf("expected status pending, got %s", n.Status)
	}
	if n.Attempts != 0 {
		t.Errorf("expected 0 attempts, got %d", n.Attempts)
	}
	if n.MaxAttempts != 5 {
		t.Errorf("expected max_attempts 5, got %d", n.MaxAttempts)
	}
}

func TestNewNotification_EmptyEventID(t *testing.T) {
	_, err := NewNotification("", ChannelEmail, "user@example.com", "Subject", "Body")
	if err == nil {
		t.Fatal("expected error for empty event_id")
	}
	if err.Error() != "event_id is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewNotification_WhatsAppChannel(t *testing.T) {
	n, err := NewNotification("event-123", ChannelWhatsApp, "group-xyz", "Invoice Summary", "Se cargó el recibo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n.Channel != ChannelWhatsApp {
		t.Errorf("expected channel whatsapp, got %s", n.Channel)
	}
	if n.Recipient != "group-xyz" {
		t.Errorf("expected recipient group-xyz, got %s", n.Recipient)
	}
}

func TestNewNotification_UnsupportedChannel(t *testing.T) {
	_, err := NewNotification("event-123", Channel("sms"), "user@example.com", "Subject", "Body")
	if err == nil {
		t.Fatal("expected error for unsupported channel")
	}
	if err.Error() != "unsupported channel" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewNotification_EmptyRecipient(t *testing.T) {
	_, err := NewNotification("event-123", ChannelEmail, "", "Subject", "Body")
	if err == nil {
		t.Fatal("expected error for empty recipient")
	}
	if err.Error() != "recipient is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewNotification_EmptySubject(t *testing.T) {
	_, err := NewNotification("event-123", ChannelEmail, "user@example.com", "", "Body")
	if err == nil {
		t.Fatal("expected error for empty subject")
	}
	if err.Error() != "subject is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewNotification_EmptyBody(t *testing.T) {
	_, err := NewNotification("event-123", ChannelEmail, "user@example.com", "Subject", "")
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if err.Error() != "body is required" {
		t.Errorf("unexpected error: %v", err)
	}
}
