package email

import (
	"strings"
	"testing"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

func TestTemplateRendererUserRegistered(t *testing.T) {
	r := NewTemplateRenderer()
	evt := entities.Event{
		Type:    entities.EventTypeUserRegistered,
		Payload: []byte(`{"user_id":"u1","email":"a@b.com","name":"Alice"}`),
	}
	subject, body, err := r.Render(evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subject != "Welcome, Alice" {
		t.Errorf("subject = %q, want %q", subject, "Welcome, Alice")
	}
	if !strings.Contains(body, "Alice") {
		t.Errorf("body should contain Alice, got %q", body)
	}
}

func TestTemplateRendererPasswordReset(t *testing.T) {
	r := NewTemplateRenderer()
	evt := entities.Event{
		Type:    entities.EventTypePasswordResetRequested,
		Payload: []byte(`{"user_id":"u1","email":"a@b.com"}`),
	}
	subject, body, err := r.Render(evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subject != "Reset your password" {
		t.Errorf("subject = %q, want %q", subject, "Reset your password")
	}
	if !strings.Contains(body, "u1") {
		t.Errorf("body should contain user ID, got %q", body)
	}
}

func TestTemplateRendererOrderPaid(t *testing.T) {
	r := NewTemplateRenderer()
	evt := entities.Event{
		Type:    entities.EventTypeOrderPaid,
		Payload: []byte(`{"order_id":"o1","user_id":"u1","amount":99.99,"currency":"USD"}`),
	}
	subject, body, err := r.Render(evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(subject, "o1") {
		t.Errorf("subject should contain order ID, got %q", subject)
	}
	if !strings.Contains(body, "99.99") {
		t.Errorf("body should contain amount, got %q", body)
	}
}

func TestTemplateRendererOrderShipped(t *testing.T) {
	r := NewTemplateRenderer()
	evt := entities.Event{
		Type:    entities.EventTypeOrderShipped,
		Payload: []byte(`{"order_id":"o1","user_id":"u1","carrier":"FedEx","tracking_number":"TRK123"}`),
	}
	subject, body, err := r.Render(evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(subject, "o1") {
		t.Errorf("subject should contain order ID, got %q", subject)
	}
	if !strings.Contains(body, "TRK123") {
		t.Errorf("body should contain tracking number, got %q", body)
	}
}

func TestTemplateRendererUnsupportedType(t *testing.T) {
	r := NewTemplateRenderer()
	evt := entities.Event{Type: "Unknown", Payload: []byte(`{}`)}
	_, _, err := r.Render(evt)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestTemplateRendererInvalidJSON(t *testing.T) {
	r := NewTemplateRenderer()
	types := []string{
		entities.EventTypeUserRegistered,
		entities.EventTypePasswordResetRequested,
		entities.EventTypeOrderPaid,
		entities.EventTypeOrderShipped,
	}
	for _, et := range types {
		t.Run(et, func(t *testing.T) {
			evt := entities.Event{Type: et, Payload: []byte(`{invalid}`)}
			_, _, err := r.Render(evt)
			if err == nil {
				t.Fatal("expected error for invalid JSON")
			}
		})
	}
}
