package usecases

import (
	"context"
	"errors"
	"testing"

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

type fakeRenderer struct {
	err error
}

func (r *fakeRenderer) Render(evt entities.Event) (string, string, error) {
	if r.err != nil {
		return "", "", r.err
	}
	return "Subject for " + evt.Type, "Body for " + evt.Type, nil
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

func TestProcessEventsSuccess(t *testing.T) {
	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-1", Type: "UserRegistered", Payload: []byte(`{"user_id":"1","email":"a@b.com","name":"Alice"}`)},
			{ID: "evt-2", Type: "OrderPaid", Payload: []byte(`{"order_id":"o1","user_id":"1","amount":99.99,"currency":"USD","email":"a@b.com"}`)},
		},
	}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, Renderer: &fakeRenderer{}, BatchSize: 10}
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

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, Renderer: &fakeRenderer{}, BatchSize: 10}
	_, err := uc.Handle(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProcessEventsBadPayloadMarksEventFailed(t *testing.T) {
	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-bad", Type: "UserRegistered", Payload: []byte(`{invalid}`)},
		},
	}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, Renderer: &fakeRenderer{}, BatchSize: 10}
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

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, Renderer: &fakeRenderer{}, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 processed, got %d", count)
	}
}

func TestProcessEventsNotificationRepoError(t *testing.T) {
	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-1", Type: "UserRegistered", Payload: []byte(`{"user_id":"1","email":"a@b.com","name":"Alice"}`)},
		},
	}
	notifRepo := &fakeNotificationRepo{err: errors.New("insert failed")}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, Renderer: &fakeRenderer{}, BatchSize: 10}
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

func TestProcessEventsRenderError(t *testing.T) {
	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-1", Type: "UserRegistered", Payload: []byte(`{"user_id":"1","email":"a@b.com","name":"Alice"}`)},
		},
	}
	notifRepo := &fakeNotificationRepo{}
	renderer := &fakeRenderer{err: errors.New("template error")}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, Renderer: renderer, BatchSize: 10}
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

func TestProcessEventsAllEventTypes(t *testing.T) {
	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "e1", Type: entities.EventTypeUserRegistered, Payload: []byte(`{"user_id":"1","email":"a@b.com","name":"A"}`)},
			{ID: "e2", Type: entities.EventTypePasswordResetRequested, Payload: []byte(`{"user_id":"1","email":"b@b.com"}`)},
			{ID: "e3", Type: entities.EventTypeOrderPaid, Payload: []byte(`{"order_id":"o1","user_id":"1","amount":10,"currency":"USD","email":"c@b.com"}`)},
			{ID: "e4", Type: entities.EventTypeOrderShipped, Payload: []byte(`{"order_id":"o1","user_id":"1","carrier":"FedEx","tracking_number":"T1","email":"d@b.com"}`)},
		},
	}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, Renderer: &fakeRenderer{}, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected 4 processed, got %d", count)
	}
}

func TestProcessEventsUnsupportedType(t *testing.T) {
	eventRepo := &fakeEventRepoForWorker{
		claimed: []entities.Event{
			{ID: "evt-1", Type: "UnknownType", Payload: []byte(`{}`)},
		},
	}
	notifRepo := &fakeNotificationRepo{}

	uc := ProcessEvents{EventRepo: eventRepo, NotificationRepo: notifRepo, Renderer: &fakeRenderer{}, BatchSize: 10}
	count, err := uc.Handle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 processed, got %d", count)
	}
	if eventRepo.statuses["evt-1"] != "failed" {
		t.Fatalf("expected event marked as failed")
	}
}

func TestExtractRecipientAllTypes(t *testing.T) {
	cases := []struct {
		name      string
		eventType string
		payload   string
		wantEmail string
	}{
		{"UserRegistered", entities.EventTypeUserRegistered, `{"user_id":"u1","email":"a@b.com","name":"A"}`, "a@b.com"},
		{"PasswordReset", entities.EventTypePasswordResetRequested, `{"user_id":"u1","email":"reset@b.com"}`, "reset@b.com"},
		{"OrderPaid", entities.EventTypeOrderPaid, `{"email":"paid@b.com"}`, "paid@b.com"},
		{"OrderShipped", entities.EventTypeOrderShipped, `{"email":"ship@b.com"}`, "ship@b.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			evt := entities.Event{Type: tc.eventType, Payload: []byte(tc.payload)}
			got, err := extractRecipient(evt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantEmail {
				t.Fatalf("email = %q, want %q", got, tc.wantEmail)
			}
		})
	}
}

func TestExtractRecipientUnsupportedType(t *testing.T) {
	evt := entities.Event{Type: "Unknown", Payload: []byte(`{}`)}
	_, err := extractRecipient(evt)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestExtractRecipientInvalidJSON(t *testing.T) {
	types := []string{
		entities.EventTypeUserRegistered,
		entities.EventTypePasswordResetRequested,
		entities.EventTypeOrderPaid,
		entities.EventTypeOrderShipped,
	}
	for _, et := range types {
		t.Run(et, func(t *testing.T) {
			evt := entities.Event{Type: et, Payload: []byte(`{invalid`)}
			_, err := extractRecipient(evt)
			if err == nil {
				t.Fatal("expected error for invalid JSON")
			}
		})
	}
}

func TestDeliverNotificationsUpdateStatusErrorOnProcessing(t *testing.T) {
	// When UpdateStatus fails during the initial "processing" transition, the notification is skipped
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
