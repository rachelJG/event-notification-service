package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rachelJG/event-notification-service/internal/application/validation"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
)

type fakeNotificationRepo struct {
	created []entities.Notification
	pending []entities.Notification
	updates []statusUpdate
	err     error
}

type statusUpdate struct {
	ID     string
	Update ports.NotificationUpdate
}

func (r *fakeNotificationRepo) Create(_ context.Context, n entities.Notification) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	r.created = append(r.created, n)
	return "notif-" + n.EventID, nil
}

func (r *fakeNotificationRepo) FindPending(_ context.Context, limit int) ([]entities.Notification, error) {
	if r.err != nil {
		return nil, r.err
	}
	if limit > len(r.pending) {
		return r.pending, nil
	}
	return r.pending[:limit], nil
}

func (r *fakeNotificationRepo) UpdateStatus(_ context.Context, id string, update ports.NotificationUpdate) error {
	if r.err != nil {
		return r.err
	}
	r.updates = append(r.updates, statusUpdate{ID: id, Update: update})
	return nil
}

type fakeEventRepoForWorker struct {
	claimed   []entities.Event
	statuses  map[string]string
	claimErr  error
	statusErr error
}

func (r *fakeEventRepoForWorker) Create(_ context.Context, _ entities.Event) (string, error) {
	return "", nil
}
func (r *fakeEventRepoForWorker) GetByID(_ context.Context, _ string) (entities.Event, error) {
	return entities.Event{}, nil
}
func (r *fakeEventRepoForWorker) ClaimPending(_ context.Context, _ int) ([]entities.Event, error) {
	if r.claimErr != nil {
		return nil, r.claimErr
	}
	return r.claimed, nil
}
func (r *fakeEventRepoForWorker) SetStatus(_ context.Context, id string, status string) error {
	if r.statusErr != nil {
		return r.statusErr
	}
	if r.statuses == nil {
		r.statuses = make(map[string]string)
	}
	r.statuses[id] = status
	return nil
}

// mustMarshalNotifications is a test helper to serialize notification specs to JSON.
func mustMarshalNotifications(t *testing.T, specs []validation.NotificationSpec) []byte {
	t.Helper()
	b, err := json.Marshal(specs)
	if err != nil {
		t.Fatalf("marshal notifications: %v", err)
	}
	return b
}

func TestProcessEventsSuccess(t *testing.T) {
	specs := []validation.NotificationSpec{
		{Channel: "email", Subject: "Welcome", Body: "Hello", Recipients: []string{"a@b.com"}},
	}
	notifJSON := mustMarshalNotifications(t, specs)

	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-1", Type: "UserRegistered", NotificationsJSON: notifJSON},
			{ID: "evt-2", Type: "OrderPaid", NotificationsJSON: notifJSON},
		},
	}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 processed, got %d", count)
	}
	if len(notifRepo.created) != 2 {
		t.Fatalf("expected 2 notifications created, got %d", len(notifRepo.created))
	}
	if notifRepo.created[0].Recipient != "a@b.com" {
		t.Errorf("expected recipient a@b.com, got %s", notifRepo.created[0].Recipient)
	}
	if notifRepo.created[0].Channel != entities.ChannelEmail {
		t.Errorf("expected channel email, got %s", notifRepo.created[0].Channel)
	}
}

func TestProcessEventsClaimError(t *testing.T) {
	eventRepo := &fakeEventRepoForWorker{claimErr: errors.New("db down")}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, BatchSize: 10}
	_, err := uc.Handle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProcessEventsBadNotificationsJSONMarksEventFailed(t *testing.T) {
	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-bad", Type: "UserRegistered", NotificationsJSON: []byte(`{invalid}`)},
		},
	}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 processed, got %d", count)
	}
	if eventRepo.statuses["evt-bad"] != "failed" {
		t.Fatalf("expected event marked as failed, got %s", eventRepo.statuses["evt-bad"])
	}
}

func TestProcessEventsNoPendingEvents(t *testing.T) {
	eventRepo := &fakeEventRepoForWorker{claimed: nil}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 processed, got %d", count)
	}
}

func TestProcessEventsNotificationRepoError(t *testing.T) {
	specs := []validation.NotificationSpec{
		{Channel: "email", Subject: "Hi", Body: "Hello", Recipients: []string{"a@b.com"}},
	}
	notifJSON := mustMarshalNotifications(t, specs)

	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-1", Type: "UserRegistered", NotificationsJSON: notifJSON},
		},
	}
	notifRepo := &fakeNotificationRepo{err: errors.New("insert failed")}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error (soft failure), got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 processed, got %d", count)
	}
	if eventRepo.statuses["evt-1"] != "failed" {
		t.Fatalf("expected event marked as failed")
	}
}

func TestProcessEventsEmailFanOut(t *testing.T) {
	specs := []validation.NotificationSpec{
		{Channel: "email", Subject: "Recibo", Body: "Su recibo", Recipients: []string{"maria@a.com", "jose@a.com", "ana@a.com"}},
	}
	notifJSON := mustMarshalNotifications(t, specs)

	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-inv", Type: entities.EventTypeInvoiceIssued, NotificationsJSON: notifJSON},
		},
	}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 notifications from fan-out, got %d", count)
	}
	if len(notifRepo.created) != 3 {
		t.Fatalf("expected 3 notifications created, got %d", len(notifRepo.created))
	}

	emails := make(map[string]bool)
	for _, n := range notifRepo.created {
		emails[n.Recipient] = true
		if n.Channel != entities.ChannelEmail {
			t.Errorf("expected email channel, got %s", n.Channel)
		}
	}
	for _, want := range []string{"maria@a.com", "jose@a.com", "ana@a.com"} {
		if !emails[want] {
			t.Errorf("missing notification for %s", want)
		}
	}
}

func TestProcessEventsWhatsAppNotification(t *testing.T) {
	specs := []validation.NotificationSpec{
		{Channel: "whatsapp", Body: "Se cargó el recibo de marzo", Recipients: []string{"group-xyz"}},
	}
	notifJSON := mustMarshalNotifications(t, specs)

	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-sum", Type: entities.EventTypeInvoiceSummary, NotificationsJSON: notifJSON},
		},
	}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 notification, got %d", count)
	}

	n := notifRepo.created[0]
	if n.Channel != entities.ChannelWhatsApp {
		t.Errorf("expected whatsapp channel, got %s", n.Channel)
	}
	if n.Recipient != "group-xyz" {
		t.Errorf("expected recipient group-xyz, got %s", n.Recipient)
	}
	if n.Body != "Se cargó el recibo de marzo" {
		t.Errorf("unexpected body: %s", n.Body)
	}
}

func TestProcessEventsMixedChannels(t *testing.T) {
	specs := []validation.NotificationSpec{
		{Channel: "email", Subject: "Recibo", Body: "Su recibo", Recipients: []string{"a@b.com", "c@d.com"}},
		{Channel: "whatsapp", Body: "Resumen mensual", Recipients: []string{"group-1"}},
	}
	notifJSON := mustMarshalNotifications(t, specs)

	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-mix", Type: entities.EventTypeInvoiceIssued, NotificationsJSON: notifJSON},
		},
	}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 notifications (2 email + 1 whatsapp), got %d", count)
	}

	var emailCount, waCount int
	for _, n := range notifRepo.created {
		switch n.Channel {
		case entities.ChannelEmail:
			emailCount++
		case entities.ChannelWhatsApp:
			waCount++
		}
	}
	if emailCount != 2 {
		t.Errorf("expected 2 email notifications, got %d", emailCount)
	}
	if waCount != 1 {
		t.Errorf("expected 1 whatsapp notification, got %d", waCount)
	}
}

func TestProcessEventsEmptyRecipientMarksEventFailed(t *testing.T) {
	specs := []validation.NotificationSpec{
		{Channel: "email", Subject: "Hi", Body: "Hello", Recipients: []string{""}},
	}
	notifJSON := mustMarshalNotifications(t, specs)

	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-1", Type: entities.EventTypeUserRegistered, NotificationsJSON: notifJSON},
		},
	}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error (soft failure), got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 processed, got %d", count)
	}
	if eventRepo.statuses["evt-1"] != "failed" {
		t.Fatalf("expected event marked as failed")
	}
}

// fakeNotificationRepoWithCallback provides fine-grained control over UpdateStatus behavior.
type fakeNotificationRepoWithCallback struct {
	pending    []entities.Notification
	updateFunc func(id string, update ports.NotificationUpdate) error
}

func (r *fakeNotificationRepoWithCallback) Create(_ context.Context, n entities.Notification) (string, error) {
	return "notif-1", nil
}

func (r *fakeNotificationRepoWithCallback) FindPending(_ context.Context, limit int) ([]entities.Notification, error) {
	return r.pending, nil
}

func (r *fakeNotificationRepoWithCallback) UpdateStatus(_ context.Context, id string, update ports.NotificationUpdate) error {
	return r.updateFunc(id, update)
}

func TestDeliverNotificationsUpdateStatusErrorOnProcessing(t *testing.T) {
	callCount := 0
	notifRepo := &fakeNotificationRepoWithCallback{
		pending: []entities.Notification{
			{ID: "n1", Recipient: "a@b.com", Subject: "Sub", Body: "Body", MaxAttempts: 5},
		},
		updateFunc: func(id string, update ports.NotificationUpdate) error {
			callCount++
			if callCount == 1 {
				return errors.New("update failed")
			}
			return nil
		},
	}
	sender := &fakeSender{}

	uc := DeliverNotifications{NotificationRepo: notifRepo, Sender: sender, BatchSize: 10}
	delivered, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delivered != 0 {
		t.Fatalf("delivered = %d, want 0 (processing update failed)", delivered)
	}
}
