package validation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

// NotificationSpec represents a client-provided notification instruction.
type NotificationSpec struct {
	Channel    string   `json:"channel"`
	From       string   `json:"from,omitempty"`
	Subject    string   `json:"subject"`
	Body       string   `json:"body"`
	Recipients []string `json:"recipients"`
}

const maxRecipientsPerNotification = 500

// ValidateEvent validates the event payload and notification specs.
// The event type is validated by the domain layer (NewEvent); this function
// validates the notification instructions provided by the client.
func ValidateEvent(eventType string, payload []byte, notifications []NotificationSpec) error {
	if len(payload) > 0 && !json.Valid(payload) {
		return apperror.InvalidArgument("invalid JSON payload", nil)
	}

	if len(notifications) == 0 {
		return apperror.InvalidArgument("at least one notification is required", nil)
	}

	for i, n := range notifications {
		if err := validateNotificationSpec(i, n); err != nil {
			return err
		}
	}

	return nil
}

func validateNotificationSpec(index int, n NotificationSpec) error {
	channel := strings.TrimSpace(n.Channel)
	if channel != "email" && channel != "whatsapp" {
		return apperror.InvalidArgument(fmt.Sprintf("notifications[%d]: unsupported channel %q", index, n.Channel), nil)
	}

	if len(n.Recipients) == 0 {
		return apperror.InvalidArgument(fmt.Sprintf("notifications[%d]: at least one recipient is required", index), nil)
	}
	if len(n.Recipients) > maxRecipientsPerNotification {
		return apperror.InvalidArgument(fmt.Sprintf("notifications[%d]: recipients exceeds maximum of %d", index, maxRecipientsPerNotification), nil)
	}

	if strings.TrimSpace(n.Body) == "" {
		return apperror.InvalidArgument(fmt.Sprintf("notifications[%d]: body is required", index), nil)
	}

	if channel == "email" {
		if strings.TrimSpace(n.Subject) == "" {
			return apperror.InvalidArgument(fmt.Sprintf("notifications[%d]: subject is required for email channel", index), nil)
		}
		if n.From != "" && !strings.Contains(n.From, "@") {
			return apperror.InvalidArgument(fmt.Sprintf("notifications[%d]: invalid from email address", index), nil)
		}
		for j, r := range n.Recipients {
			if !strings.Contains(r, "@") {
				return apperror.InvalidArgument(fmt.Sprintf("notifications[%d].recipients[%d]: invalid email address", index, j), nil)
			}
		}
	}

	if channel == "whatsapp" {
		for j, r := range n.Recipients {
			if strings.TrimSpace(r) == "" {
				return apperror.InvalidArgument(fmt.Sprintf("notifications[%d].recipients[%d]: empty group ID", index, j), nil)
			}
		}
	}

	return nil
}
