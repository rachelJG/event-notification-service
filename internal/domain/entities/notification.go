package entities

import (
	"errors"
	"strings"
	"time"
)

// NotificationStatus is a value object representing the lifecycle state of a notification.
type NotificationStatus string

const (
	NotificationStatusPending    NotificationStatus = "pending"
	NotificationStatusProcessing NotificationStatus = "processing"
	NotificationStatusDelivered  NotificationStatus = "delivered"
	NotificationStatusFailed     NotificationStatus = "failed"
)

// Channel is a value object representing the delivery channel.
type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelWhatsApp Channel = "whatsapp"
)

// Notification represents a message to be delivered through a channel.
type Notification struct {
	ID          string
	EventID     string
	Channel     Channel
	From        string // sender address (email only); empty means use service default
	Recipient   string
	Subject     string
	Body        string
	Status      NotificationStatus
	Attempts    int
	MaxAttempts int
	LastError   string
	NextRetryAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewNotification constructs a Notification enforcing domain invariants.
func NewNotification(eventID string, channel Channel, from, recipient, subject, body string) (Notification, error) {
	if strings.TrimSpace(eventID) == "" {
		return Notification{}, errors.New("event_id is required")
	}
	if channel != ChannelEmail && channel != ChannelWhatsApp {
		return Notification{}, errors.New("unsupported channel")
	}
	if strings.TrimSpace(recipient) == "" {
		return Notification{}, errors.New("recipient is required")
	}
	if channel == ChannelEmail && strings.TrimSpace(subject) == "" {
		return Notification{}, errors.New("subject is required for email channel")
	}
	if strings.TrimSpace(body) == "" {
		return Notification{}, errors.New("body is required")
	}
	return Notification{
		EventID:     eventID,
		Channel:     channel,
		From:        from,
		Recipient:   recipient,
		Subject:     subject,
		Body:        body,
		Status:      NotificationStatusPending,
		Attempts:    0,
		MaxAttempts: 5,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}, nil
}
