package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

func ValidateEvent(eventType string, payload []byte) error {
	switch eventType {
	case entities.EventTypeUserRegistered:
		var body entities.UserRegisteredPayload
		if err := json.Unmarshal(payload, &body); err != nil {
			return errors.New("invalid UserRegistered payload")
		}
		if strings.TrimSpace(body.UserID) == "" || strings.TrimSpace(body.Email) == "" || strings.TrimSpace(body.Name) == "" {
			return errors.New("UserRegistered requires user_id, email, and name")
		}
		if !strings.Contains(body.Email, "@") {
			return errors.New("UserRegistered requires a valid email")
		}
		return nil
	case entities.EventTypePasswordResetRequested:
		var body entities.PasswordResetRequestedPayload
		if err := json.Unmarshal(payload, &body); err != nil {
			return errors.New("invalid PasswordResetRequested payload")
		}
		if strings.TrimSpace(body.UserID) == "" || strings.TrimSpace(body.Email) == "" {
			return errors.New("PasswordResetRequested requires user_id and email")
		}
		if !strings.Contains(body.Email, "@") {
			return errors.New("PasswordResetRequested requires a valid email")
		}
		return nil
	case entities.EventTypeOrderPaid:
		var body entities.OrderPaidPayload
		if err := json.Unmarshal(payload, &body); err != nil {
			return errors.New("invalid OrderPaid payload")
		}
		if strings.TrimSpace(body.OrderID) == "" || strings.TrimSpace(body.UserID) == "" || strings.TrimSpace(body.Currency) == "" {
			return errors.New("OrderPaid requires order_id, user_id, and currency")
		}
		if body.Amount <= 0 {
			return errors.New("OrderPaid requires amount > 0")
		}
		return nil
	case entities.EventTypeOrderShipped:
		var body entities.OrderShippedPayload
		if err := json.Unmarshal(payload, &body); err != nil {
			return errors.New("invalid OrderShipped payload")
		}
		if strings.TrimSpace(body.OrderID) == "" || strings.TrimSpace(body.UserID) == "" || strings.TrimSpace(body.Carrier) == "" || strings.TrimSpace(body.TrackingNumber) == "" {
			return errors.New("OrderShipped requires order_id, user_id, carrier, and tracking_number")
		}
		return nil
	case entities.EventTypeInvoiceIssued:
		var body entities.InvoiceIssuedPayload
		if err := json.Unmarshal(payload, &body); err != nil {
			return errors.New("invalid InvoiceIssued payload")
		}
		if strings.TrimSpace(body.CondominiumID) == "" || strings.TrimSpace(body.CondominiumName) == "" {
			return errors.New("InvoiceIssued requires condominium_id and condominium_name")
		}
		if strings.TrimSpace(body.InvoiceMonth) == "" || strings.TrimSpace(body.DueDate) == "" {
			return errors.New("InvoiceIssued requires invoice_month and due_date")
		}
		if strings.TrimSpace(body.Currency) == "" {
			return errors.New("InvoiceIssued requires currency")
		}
		if len(body.Recipients) == 0 {
			return errors.New("InvoiceIssued requires at least one recipient")
		}
		if len(body.Recipients) > 500 {
			return errors.New("InvoiceIssued recipients exceeds maximum of 500")
		}
		for i, r := range body.Recipients {
			if strings.TrimSpace(r.Email) == "" || strings.TrimSpace(r.Name) == "" || strings.TrimSpace(r.UnitCode) == "" {
				return fmt.Errorf("InvoiceIssued recipient[%d] requires email, name, and unit_code", i)
			}
			if !strings.Contains(r.Email, "@") {
				return fmt.Errorf("InvoiceIssued recipient[%d] has invalid email", i)
			}
			if r.Amount <= 0 {
				return fmt.Errorf("InvoiceIssued recipient[%d] requires amount > 0", i)
			}
		}
		return nil
	case entities.EventTypeInvoiceSummary:
		var body entities.InvoiceSummaryPayload
		if err := json.Unmarshal(payload, &body); err != nil {
			return errors.New("invalid InvoiceSummary payload")
		}
		if strings.TrimSpace(body.CondominiumID) == "" || strings.TrimSpace(body.CondominiumName) == "" {
			return errors.New("InvoiceSummary requires condominium_id and condominium_name")
		}
		if strings.TrimSpace(body.InvoiceMonth) == "" {
			return errors.New("InvoiceSummary requires invoice_month")
		}
		if body.TotalUnits <= 0 {
			return errors.New("InvoiceSummary requires total_units > 0")
		}
		if body.TotalAmount <= 0 {
			return errors.New("InvoiceSummary requires total_amount > 0")
		}
		if strings.TrimSpace(body.Currency) == "" {
			return errors.New("InvoiceSummary requires currency")
		}
		if strings.TrimSpace(body.WhatsAppGroupID) == "" {
			return errors.New("InvoiceSummary requires whatsapp_group_id")
		}
		if strings.TrimSpace(body.Message) == "" {
			return errors.New("InvoiceSummary requires message")
		}
		return nil
	default:
		return errors.New("unsupported event_type")
	}
}
