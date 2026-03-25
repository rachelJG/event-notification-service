package ports

import "context"

// WhatsAppSender sends messages to WhatsApp groups or individuals.
type WhatsAppSender interface {
	SendToGroup(ctx context.Context, groupID, message string) error
}
