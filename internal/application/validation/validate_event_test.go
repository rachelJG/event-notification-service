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

func TestValidateEventInvoiceIssued(t *testing.T) {
	payload := []byte(`{
		"condominium_id":"c1","condominium_name":"Residencias Sol",
		"invoice_month":"2026-03","due_date":"2026-04-10","currency":"USD",
		"recipients":[
			{"email":"maria@email.com","name":"María García","unit_code":"1-A","amount":150.00},
			{"email":"jose@email.com","name":"José López","unit_code":"2-B","amount":200.00}
		]
	}`)
	if err := ValidateEvent("InvoiceIssued", payload); err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}
}

func TestValidateEventInvoiceIssuedInvalidJSON(t *testing.T) {
	if err := ValidateEvent("InvoiceIssued", []byte(`not-json`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateEventInvoiceIssuedMissingCondominiumID(t *testing.T) {
	payload := []byte(`{
		"condominium_id":"","condominium_name":"Sol",
		"invoice_month":"2026-03","due_date":"2026-04-10","currency":"USD",
		"recipients":[{"email":"a@b.com","name":"A","unit_code":"1-A","amount":100}]
	}`)
	if err := ValidateEvent("InvoiceIssued", payload); err == nil {
		t.Fatal("expected validation error for missing condominium_id")
	}
}

func TestValidateEventInvoiceIssuedEmptyRecipients(t *testing.T) {
	payload := []byte(`{
		"condominium_id":"c1","condominium_name":"Sol",
		"invoice_month":"2026-03","due_date":"2026-04-10","currency":"USD",
		"recipients":[]
	}`)
	if err := ValidateEvent("InvoiceIssued", payload); err == nil {
		t.Fatal("expected validation error for empty recipients")
	}
}

func TestValidateEventInvoiceIssuedInvalidRecipientEmail(t *testing.T) {
	payload := []byte(`{
		"condominium_id":"c1","condominium_name":"Sol",
		"invoice_month":"2026-03","due_date":"2026-04-10","currency":"USD",
		"recipients":[{"email":"not-email","name":"A","unit_code":"1-A","amount":100}]
	}`)
	if err := ValidateEvent("InvoiceIssued", payload); err == nil {
		t.Fatal("expected validation error for invalid recipient email")
	}
}

func TestValidateEventInvoiceIssuedRecipientAmountZero(t *testing.T) {
	payload := []byte(`{
		"condominium_id":"c1","condominium_name":"Sol",
		"invoice_month":"2026-03","due_date":"2026-04-10","currency":"USD",
		"recipients":[{"email":"a@b.com","name":"A","unit_code":"1-A","amount":0}]
	}`)
	if err := ValidateEvent("InvoiceIssued", payload); err == nil {
		t.Fatal("expected validation error for amount = 0")
	}
}

func TestValidateEventInvoiceSummary(t *testing.T) {
	payload := []byte(`{
		"condominium_id":"c1","condominium_name":"Residencias Sol",
		"invoice_month":"2026-03","total_units":180,"total_amount":27000.00,
		"currency":"USD","whatsapp_group_id":"group-xyz",
		"message":"Se cargó el recibo de marzo 2026. Total: $27,000"
	}`)
	if err := ValidateEvent("InvoiceSummary", payload); err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}
}

func TestValidateEventInvoiceSummaryInvalidJSON(t *testing.T) {
	if err := ValidateEvent("InvoiceSummary", []byte(`not-json`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidateEventInvoiceSummaryMissingWhatsAppGroupID(t *testing.T) {
	payload := []byte(`{
		"condominium_id":"c1","condominium_name":"Sol",
		"invoice_month":"2026-03","total_units":10,"total_amount":1000,
		"currency":"USD","whatsapp_group_id":"","message":"msg"
	}`)
	if err := ValidateEvent("InvoiceSummary", payload); err == nil {
		t.Fatal("expected validation error for missing whatsapp_group_id")
	}
}

func TestValidateEventInvoiceSummaryMissingMessage(t *testing.T) {
	payload := []byte(`{
		"condominium_id":"c1","condominium_name":"Sol",
		"invoice_month":"2026-03","total_units":10,"total_amount":1000,
		"currency":"USD","whatsapp_group_id":"g1","message":""
	}`)
	if err := ValidateEvent("InvoiceSummary", payload); err == nil {
		t.Fatal("expected validation error for missing message")
	}
}

func TestValidateEventInvoiceSummaryTotalUnitsZero(t *testing.T) {
	payload := []byte(`{
		"condominium_id":"c1","condominium_name":"Sol",
		"invoice_month":"2026-03","total_units":0,"total_amount":1000,
		"currency":"USD","whatsapp_group_id":"g1","message":"msg"
	}`)
	if err := ValidateEvent("InvoiceSummary", payload); err == nil {
		t.Fatal("expected validation error for total_units = 0")
	}
}

func TestValidateEventEmptyType(t *testing.T) {
	if err := ValidateEvent("", []byte(`{}`)); err == nil {
		t.Fatal("expected validation error for empty type")
	}
}
