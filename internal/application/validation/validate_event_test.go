package validation

import "testing"

func TestValidateEventUserRegistered(t *testing.T) {
	payload := []byte(`{"user_id":"123","email":"user@example.com","name":"Jane"}`)
	if err := ValidateEvent("UserRegistered", payload); err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}
}

func TestValidateEventUserRegisteredMissingFields(t *testing.T) {
	payload := []byte(`{"user_id":"","email":"user@example.com","name":"Jane"}`)
	if err := ValidateEvent("UserRegistered", payload); err == nil {
		t.Fatalf("expected validation error for missing user_id")
	}
}

func TestValidateEventUnsupportedType(t *testing.T) {
	payload := []byte(`{"foo":"bar"}`)
	if err := ValidateEvent("Unknown", payload); err == nil {
		t.Fatalf("expected validation error for unsupported type")
	}
}
