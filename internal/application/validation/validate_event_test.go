package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validEmailNotification() NotificationSpec {
	return NotificationSpec{
		Channel:    "email",
		Subject:    "Test Subject",
		Body:       "<p>Hello</p>",
		Recipients: []string{"user@example.com"},
	}
}

func validWhatsAppNotification() NotificationSpec {
	return NotificationSpec{
		Channel:    "whatsapp",
		Body:       "Hello group",
		Recipients: []string{"group-123"},
	}
}

func TestValidateEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		eventType     string
		payload       []byte
		notifications []NotificationSpec
		wantErr       string
	}{
		{
			name:          "valid email notification",
			eventType:     "UserRegistered",
			payload:       []byte(`{"user_id":"123"}`),
			notifications: []NotificationSpec{validEmailNotification()},
		},
		{
			name:          "valid whatsapp notification",
			eventType:     "InvoiceSummary",
			payload:       []byte(`{"condominium_id":"c1"}`),
			notifications: []NotificationSpec{validWhatsAppNotification()},
		},
		{
			name:      "valid mixed channels",
			eventType: "InvoiceIssued",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				validEmailNotification(),
				validWhatsAppNotification(),
			},
		},
		{
			name:          "nil payload is accepted",
			eventType:     "OrderPaid",
			payload:       nil,
			notifications: []NotificationSpec{validEmailNotification()},
		},
		{
			name:          "empty payload is accepted",
			eventType:     "OrderPaid",
			payload:       []byte{},
			notifications: []NotificationSpec{validEmailNotification()},
		},
		{
			name:          "invalid JSON payload",
			eventType:     "UserRegistered",
			payload:       []byte(`not-json`),
			notifications: []NotificationSpec{validEmailNotification()},
			wantErr:       "invalid JSON payload",
		},
		{
			name:          "empty notifications",
			eventType:     "UserRegistered",
			payload:       []byte(`{}`),
			notifications: []NotificationSpec{},
			wantErr:       "at least one notification is required",
		},
		{
			name:          "nil notifications",
			eventType:     "UserRegistered",
			payload:       []byte(`{}`),
			notifications: nil,
			wantErr:       "at least one notification is required",
		},
		{
			name:      "unsupported channel",
			eventType: "UserRegistered",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "sms", Body: "hello", Recipients: []string{"123"}},
			},
			wantErr: `notifications[0]: unsupported channel "sms"`,
		},
		{
			name:      "empty recipients",
			eventType: "UserRegistered",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "email", Subject: "Hi", Body: "hello", Recipients: []string{}},
			},
			wantErr: "notifications[0]: at least one recipient is required",
		},
		{
			name:      "empty body",
			eventType: "UserRegistered",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "email", Subject: "Hi", Body: "", Recipients: []string{"a@b.com"}},
			},
			wantErr: "notifications[0]: body is required",
		},
		{
			name:      "email missing subject",
			eventType: "UserRegistered",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "email", Subject: "", Body: "hello", Recipients: []string{"a@b.com"}},
			},
			wantErr: "notifications[0]: subject is required for email channel",
		},
		{
			name:      "email invalid recipient",
			eventType: "UserRegistered",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "email", Subject: "Hi", Body: "hello", Recipients: []string{"not-email"}},
			},
			wantErr: "notifications[0].recipients[0]: invalid email address",
		},
		{
			name:      "whatsapp empty group ID",
			eventType: "InvoiceSummary",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "whatsapp", Body: "hello", Recipients: []string{""}},
			},
			wantErr: "notifications[0].recipients[0]: empty group ID",
		},
		{
			name:      "whatsapp subject is optional",
			eventType: "InvoiceSummary",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "whatsapp", Subject: "", Body: "hello", Recipients: []string{"group-1"}},
			},
		},
		{
			name:      "email with HTML body",
			eventType: "UserRegistered",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "email", Subject: "Welcome", Body: "<h1>Welcome!</h1><p>Thanks for joining</p>", Recipients: []string{"user@test.com"}},
			},
		},
		{
			name:      "multiple email recipients (fan-out)",
			eventType: "InvoiceIssued",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "email", Subject: "Recibo", Body: "Su recibo", Recipients: []string{"a@b.com", "c@d.com", "e@f.com"}},
			},
		},
		{
			name:      "email with valid from",
			eventType: "UserRegistered",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "email", From: "noreply@client.com", Subject: "Welcome", Body: "Hello", Recipients: []string{"a@b.com"}},
			},
		},
		{
			name:      "email with invalid from",
			eventType: "UserRegistered",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "email", From: "not-an-email", Subject: "Welcome", Body: "Hello", Recipients: []string{"a@b.com"}},
			},
			wantErr: "notifications[0]: invalid from email address",
		},
		{
			name:      "email with empty from uses default",
			eventType: "UserRegistered",
			payload:   []byte(`{}`),
			notifications: []NotificationSpec{
				{Channel: "email", From: "", Subject: "Welcome", Body: "Hello", Recipients: []string{"a@b.com"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateEvent(tt.eventType, tt.payload, tt.notifications)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidateEvent_TooManyRecipients(t *testing.T) {
	t.Parallel()

	recipients := make([]string, 501)
	for i := range recipients {
		recipients[i] = "user@example.com"
	}

	err := ValidateEvent("UserRegistered", []byte(`{}`), []NotificationSpec{
		{Channel: "email", Subject: "Hi", Body: "hello", Recipients: recipients},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum of 500")
}
