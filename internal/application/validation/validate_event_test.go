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
		t.Fatal("expected validation error for missing user_id")
	}
}

func TestValidateEventUserRegisteredInvalidEmail(t *testing.T) {
	payload := []byte(`{"user_id":"1","email":"not-an-email","name":"Jane"}`)
	if err := ValidateEvent("UserRegistered", payload); err == nil {
		t.Fatal("expected validation error for invalid email")
	}
}

func TestValidateEventUserRegisteredInvalidJSON(t *testing.T) {
	if err := ValidateEvent("UserRegistered", []byte(`not-json`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateEventPasswordResetRequested(t *testing.T) {
	payload := []byte(`{"user_id":"1","email":"user@example.com"}`)
	if err := ValidateEvent("PasswordResetRequested", payload); err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}
}

func TestValidateEventPasswordResetRequestedMissingFields(t *testing.T) {
	payload := []byte(`{"user_id":"","email":"user@example.com"}`)
	if err := ValidateEvent("PasswordResetRequested", payload); err == nil {
		t.Fatal("expected validation error for missing user_id")
	}
}

func TestValidateEventPasswordResetRequestedInvalidEmail(t *testing.T) {
	payload := []byte(`{"user_id":"1","email":"not-an-email"}`)
	if err := ValidateEvent("PasswordResetRequested", payload); err == nil {
		t.Fatal("expected validation error for invalid email")
	}
}

func TestValidateEventPasswordResetRequestedInvalidJSON(t *testing.T) {
	if err := ValidateEvent("PasswordResetRequested", []byte(`not-json`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateEventOrderPaid(t *testing.T) {
	payload := []byte(`{"order_id":"o1","user_id":"u1","amount":9.99,"currency":"USD"}`)
	if err := ValidateEvent("OrderPaid", payload); err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}
}

func TestValidateEventOrderPaidAmountZero(t *testing.T) {
	payload := []byte(`{"order_id":"o1","user_id":"u1","amount":0,"currency":"USD"}`)
	if err := ValidateEvent("OrderPaid", payload); err == nil {
		t.Fatal("expected validation error for amount = 0")
	}
}

func TestValidateEventOrderPaidAmountNegative(t *testing.T) {
	payload := []byte(`{"order_id":"o1","user_id":"u1","amount":-1,"currency":"USD"}`)
	if err := ValidateEvent("OrderPaid", payload); err == nil {
		t.Fatal("expected validation error for negative amount")
	}
}

func TestValidateEventOrderPaidMissingFields(t *testing.T) {
	payload := []byte(`{"order_id":"","user_id":"u1","amount":9.99,"currency":"USD"}`)
	if err := ValidateEvent("OrderPaid", payload); err == nil {
		t.Fatal("expected validation error for missing order_id")
	}
}

func TestValidateEventOrderPaidInvalidJSON(t *testing.T) {
	if err := ValidateEvent("OrderPaid", []byte(`not-json`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateEventOrderShipped(t *testing.T) {
	payload := []byte(`{"order_id":"o1","user_id":"u1","carrier":"DHL","tracking_number":"TRK123"}`)
	if err := ValidateEvent("OrderShipped", payload); err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}
}

func TestValidateEventOrderShippedMissingFields(t *testing.T) {
	payload := []byte(`{"order_id":"o1","user_id":"u1","carrier":"","tracking_number":"TRK123"}`)
	if err := ValidateEvent("OrderShipped", payload); err == nil {
		t.Fatal("expected validation error for missing carrier")
	}
}

func TestValidateEventOrderShippedInvalidJSON(t *testing.T) {
	if err := ValidateEvent("OrderShipped", []byte(`not-json`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateEventUnsupportedType(t *testing.T) {
	if err := ValidateEvent("Unknown", []byte(`{"foo":"bar"}`)); err == nil {
		t.Fatal("expected validation error for unsupported type")
	}
}

func TestValidateEventEmptyType(t *testing.T) {
	if err := ValidateEvent("", []byte(`{}`)); err == nil {
		t.Fatal("expected validation error for empty type")
	}
}
