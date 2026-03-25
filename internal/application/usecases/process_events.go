package usecases

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
)

type ProcessEvents struct {
	EventRepo        ports.EventRepository
	NotificationRepo ports.NotificationRepository
	Renderer         ports.EmailRenderer
	BatchSize        int
}

func (uc ProcessEvents) Handle(ctx context.Context) (int, error) {
	events, err := uc.EventRepo.ClaimPending(ctx, uc.BatchSize)
	if err != nil {
		return 0, fmt.Errorf("claim pending events: %w", err)
	}

	processed := 0
	for _, evt := range events {
		n, err := uc.processEvent(ctx, evt)
		if err != nil {
			_ = uc.EventRepo.SetStatus(ctx, evt.ID, "failed")
			continue
		}
		processed += n
	}

	return processed, nil
}

// processEvent dispatches event processing by type, returning the number of
// notifications created.
func (uc ProcessEvents) processEvent(ctx context.Context, evt entities.Event) (int, error) {
	switch evt.Type {
	case entities.EventTypeInvoiceIssued:
		return uc.processInvoiceIssued(ctx, evt)
	case entities.EventTypeInvoiceSummary:
		return uc.processInvoiceSummary(ctx, evt)
	default:
		return uc.processSingleRecipient(ctx, evt)
	}
}

// processSingleRecipient handles event types that produce exactly one email
// notification (UserRegistered, PasswordResetRequested, OrderPaid, OrderShipped).
func (uc ProcessEvents) processSingleRecipient(ctx context.Context, evt entities.Event) (int, error) {
	recipient, err := extractRecipient(evt)
	if err != nil {
		return 0, err
	}

	subject, body, err := uc.Renderer.Render(evt)
	if err != nil {
		return 0, err
	}

	n, err := entities.NewNotification(evt.ID, entities.ChannelEmail, recipient, subject, body)
	if err != nil {
		return 0, err
	}

	if _, err := uc.NotificationRepo.Create(ctx, n); err != nil {
		return 0, err
	}

	return 1, nil
}

// processInvoiceIssued fans out a single InvoiceIssued event into one email
// notification per recipient.
func (uc ProcessEvents) processInvoiceIssued(ctx context.Context, evt entities.Event) (int, error) {
	var payload entities.InvoiceIssuedPayload
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return 0, fmt.Errorf("unmarshal InvoiceIssued payload: %w", err)
	}

	created := 0
	for _, r := range payload.Recipients {
		subject, body := renderInvoiceEmail(payload, r)

		n, err := entities.NewNotification(evt.ID, entities.ChannelEmail, r.Email, subject, body)
		if err != nil {
			return created, err
		}

		if _, err := uc.NotificationRepo.Create(ctx, n); err != nil {
			return created, err
		}
		created++
	}

	return created, nil
}

// renderInvoiceEmail produces subject and body for a single invoice recipient.
func renderInvoiceEmail(p entities.InvoiceIssuedPayload, r entities.InvoiceRecipient) (subject, body string) {
	subject = fmt.Sprintf("Recibo %s - %s", p.InvoiceMonth, p.CondominiumName)
	body = fmt.Sprintf(
		"%s (%s),\n\nSe ha cargado su recibo correspondiente al mes %s.\nMonto: %.2f %s\nFecha de vencimiento: %s\n\nCondominio: %s",
		r.Name, r.UnitCode, p.InvoiceMonth, r.Amount, p.Currency, p.DueDate, p.CondominiumName,
	)
	return subject, body
}

// processInvoiceSummary creates a single WhatsApp notification for the group.
func (uc ProcessEvents) processInvoiceSummary(ctx context.Context, evt entities.Event) (int, error) {
	var payload entities.InvoiceSummaryPayload
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return 0, fmt.Errorf("unmarshal InvoiceSummary payload: %w", err)
	}

	subject := fmt.Sprintf("Resumen recibo %s", payload.InvoiceMonth)
	n, err := entities.NewNotification(evt.ID, entities.ChannelWhatsApp, payload.WhatsAppGroupID, subject, payload.Message)
	if err != nil {
		return 0, err
	}

	if _, err := uc.NotificationRepo.Create(ctx, n); err != nil {
		return 0, err
	}

	return 1, nil
}

// extractRecipient resolves the email address from the event payload.
// All single-recipient event types include an "email" JSON field.
func extractRecipient(evt entities.Event) (string, error) {
	var p struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		return "", fmt.Errorf("unmarshal payload for recipient: %w", err)
	}
	if p.Email == "" {
		return "", fmt.Errorf("no email in payload for event type: %s", evt.Type)
	}
	return p.Email, nil
}
