package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

type fakeSender struct {
	sent []sentEmail
	err  error
}

type fakeWhatsAppSender struct {
	sent []sentWhatsApp
	err  error
}

type sentWhatsApp struct {
	GroupID, Message string
}

func (s *fakeWhatsAppSender) SendToGroup(_ context.Context, groupID, message string) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, sentWhatsApp{GroupID: groupID, Message: message})
	return nil
}

type sentEmail struct {
	From, To, Subject, Body string
}

func (s *fakeSender) Send(_ context.Context, from, to, subject, body string) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, sentEmail{From: from, To: to, Subject: subject, Body: body})
	return nil
}

func TestDeliverNotificationsSuccess(t *testing.T) {
	notifRepo := &fakeNotificationRepo{
		pending: []entities.Notification{
			{ID: "n1", EventID: "e1", Channel: entities.ChannelEmail, Recipient: "a@b.com", Subject: "Hi", Body: "Hello", Status: entities.NotificationStatusPending, MaxAttempts: 5},
		},
	}
	sender := &fakeSender{}

	uc := DeliverNotifications{NotificationRepo: notifRepo, Sender: sender, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 delivered, got %d", count)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(sender.sent))
	}
	if sender.sent[0].To != "a@b.com" {
		t.Errorf("expected to a@b.com, got %s", sender.sent[0].To)
	}

	// Verify status transitions: processing → delivered
	if len(notifRepo.updates) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(notifRepo.updates))
	}
	if notifRepo.updates[0].Update.Status != entities.NotificationStatusProcessing {
		t.Errorf("expected first update to processing, got %s", notifRepo.updates[0].Update.Status)
	}
	if notifRepo.updates[1].Update.Status != entities.NotificationStatusDelivered {
		t.Errorf("expected second update to delivered, got %s", notifRepo.updates[1].Update.Status)
	}
}

func TestDeliverNotificationsSendError(t *testing.T) {
	notifRepo := &fakeNotificationRepo{
		pending: []entities.Notification{
			{ID: "n1", EventID: "e1", Channel: entities.ChannelEmail, Recipient: "a@b.com", Subject: "Hi", Body: "Hello", Attempts: 0, MaxAttempts: 5},
		},
	}
	sender := &fakeSender{err: errors.New("smtp error")}

	uc := DeliverNotifications{NotificationRepo: notifRepo, Sender: sender, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error (soft failure), got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 delivered, got %d", count)
	}

	// Should go back to pending (attempt 1 < maxAttempts 5)
	lastUpdate := notifRepo.updates[len(notifRepo.updates)-1]
	if lastUpdate.Update.Status != entities.NotificationStatusPending {
		t.Errorf("expected status pending after retry, got %s", lastUpdate.Update.Status)
	}
	if lastUpdate.Update.LastError != "smtp error" {
		t.Errorf("expected last_error 'smtp error', got %s", lastUpdate.Update.LastError)
	}
}

func TestDeliverNotificationsMaxRetriesExceeded(t *testing.T) {
	notifRepo := &fakeNotificationRepo{
		pending: []entities.Notification{
			{ID: "n1", EventID: "e1", Channel: entities.ChannelEmail, Recipient: "a@b.com", Subject: "Hi", Body: "Hello", Attempts: 4, MaxAttempts: 5},
		},
	}
	sender := &fakeSender{err: errors.New("smtp error")}

	uc := DeliverNotifications{NotificationRepo: notifRepo, Sender: sender, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 delivered, got %d", count)
	}

	lastUpdate := notifRepo.updates[len(notifRepo.updates)-1]
	if lastUpdate.Update.Status != entities.NotificationStatusFailed {
		t.Errorf("expected status failed after max retries, got %s", lastUpdate.Update.Status)
	}
}

func TestDeliverNotificationsFindPendingError(t *testing.T) {
	notifRepo := &fakeNotificationRepo{err: errors.New("db down")}
	sender := &fakeSender{}

	uc := DeliverNotifications{NotificationRepo: notifRepo, Sender: sender, BatchSize: 10}
	_, err := uc.Handle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeliverNotificationsNoPending(t *testing.T) {
	notifRepo := &fakeNotificationRepo{pending: nil}
	sender := &fakeSender{}

	uc := DeliverNotifications{NotificationRepo: notifRepo, Sender: sender, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 delivered, got %d", count)
	}
}

func TestDeliverNotificationsWhatsAppSuccess(t *testing.T) {
	notifRepo := &fakeNotificationRepo{
		pending: []entities.Notification{
			{ID: "n1", EventID: "e1", Channel: entities.ChannelWhatsApp, Recipient: "group-xyz", Subject: "Summary", Body: "Recibo marzo", Status: entities.NotificationStatusPending, MaxAttempts: 5},
		},
	}
	sender := &fakeSender{}
	whatsApp := &fakeWhatsAppSender{}

	uc := DeliverNotifications{NotificationRepo: notifRepo, Sender: sender, WhatsAppSender: whatsApp, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 delivered, got %d", count)
	}
	if len(sender.sent) != 0 {
		t.Errorf("expected no emails sent, got %d", len(sender.sent))
	}
	if len(whatsApp.sent) != 1 {
		t.Fatalf("expected 1 WhatsApp message sent, got %d", len(whatsApp.sent))
	}
	if whatsApp.sent[0].GroupID != "group-xyz" {
		t.Errorf("expected group-xyz, got %s", whatsApp.sent[0].GroupID)
	}
	if whatsApp.sent[0].Message != "Recibo marzo" {
		t.Errorf("expected message 'Recibo marzo', got %s", whatsApp.sent[0].Message)
	}
}

func TestDeliverNotificationsWhatsAppSendError(t *testing.T) {
	notifRepo := &fakeNotificationRepo{
		pending: []entities.Notification{
			{ID: "n1", EventID: "e1", Channel: entities.ChannelWhatsApp, Recipient: "group-xyz", Subject: "Summary", Body: "Msg", Attempts: 0, MaxAttempts: 5},
		},
	}
	whatsApp := &fakeWhatsAppSender{err: errors.New("whatsapp api error")}

	uc := DeliverNotifications{NotificationRepo: notifRepo, Sender: &fakeSender{}, WhatsAppSender: whatsApp, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error (soft failure), got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 delivered, got %d", count)
	}
	lastUpdate := notifRepo.updates[len(notifRepo.updates)-1]
	if lastUpdate.Update.Status != entities.NotificationStatusPending {
		t.Errorf("expected status pending after retry, got %s", lastUpdate.Update.Status)
	}
	if lastUpdate.Update.LastError != "whatsapp api error" {
		t.Errorf("expected last_error 'whatsapp api error', got %s", lastUpdate.Update.LastError)
	}
}

func TestDeliverNotificationsWhatsAppNotConfigured(t *testing.T) {
	notifRepo := &fakeNotificationRepo{
		pending: []entities.Notification{
			{ID: "n1", EventID: "e1", Channel: entities.ChannelWhatsApp, Recipient: "group-xyz", Subject: "Summary", Body: "Msg", Attempts: 0, MaxAttempts: 5},
		},
	}

	uc := DeliverNotifications{NotificationRepo: notifRepo, Sender: &fakeSender{}, WhatsAppSender: nil, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error (soft failure), got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 delivered, got %d", count)
	}
}

func TestDeliverNotificationsUnsupportedChannel(t *testing.T) {
	notifRepo := &fakeNotificationRepo{
		pending: []entities.Notification{
			{ID: "n1", EventID: "e1", Channel: entities.Channel("sms"), Recipient: "+123", Subject: "", Body: "Hello", Attempts: 0, MaxAttempts: 5},
		},
	}

	uc := DeliverNotifications{NotificationRepo: notifRepo, Sender: &fakeSender{}, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error (soft failure), got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 delivered, got %d", count)
	}
	lastUpdate := notifRepo.updates[len(notifRepo.updates)-1]
	if lastUpdate.Update.Status != entities.NotificationStatusPending {
		t.Errorf("expected status pending after retry, got %s", lastUpdate.Update.Status)
	}
}

func TestRetryDelay(t *testing.T) {
	if d := retryDelay(0); d != 1*time.Second {
		t.Errorf("expected 1s for attempt 0, got %v", d)
	}
	if d := retryDelay(1); d != 2*time.Second {
		t.Errorf("expected 2s for attempt 1, got %v", d)
	}
	if d := retryDelay(3); d != 8*time.Second {
		t.Errorf("expected 8s for attempt 3, got %v", d)
	}
}
