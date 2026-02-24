//go:build integration

package tests

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/application/usecases"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/email"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/postgres"
)

// fakeEmailSender records sent emails instead of sending them via SMTP.
type fakeEmailSender struct {
	sent []sentEmail
	err  error
}

type sentEmail struct {
	To      string
	Subject string
	Body    string
}

func (s *fakeEmailSender) Send(_ context.Context, to, subject, body string) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, sentEmail{To: to, Subject: subject, Body: body})
	return nil
}

// setupWorkerSchema ensures both events and notifications tables exist with all required columns.
func setupWorkerSchema(t *testing.T, pool *pgxpool.Pool, ctx context.Context) {
	t.Helper()

	// Ensure notifications table and types exist
	_, err := pool.Exec(ctx, `
		DO $$ BEGIN
			CREATE TYPE notification_status AS ENUM ('pending', 'processing', 'delivered', 'failed');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$;
		DO $$ BEGIN
			CREATE TYPE notification_channel AS ENUM ('email');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$;
	`)
	if err != nil {
		t.Fatalf("ensure notification types: %v", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notifications (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			event_id      UUID NOT NULL REFERENCES events(id),
			channel       notification_channel NOT NULL,
			recipient     TEXT NOT NULL,
			subject       TEXT NOT NULL,
			body          TEXT NOT NULL,
			status        notification_status NOT NULL DEFAULT 'pending',
			attempts      INT NOT NULL DEFAULT 0,
			max_attempts  INT NOT NULL DEFAULT 5,
			last_error    TEXT,
			next_retry_at TIMESTAMPTZ,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		t.Fatalf("ensure notifications table: %v", err)
	}

	// Ensure status column exists on events
	_, _ = pool.Exec(ctx, `ALTER TABLE events ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'accepted'`)

	// Truncate both tables
	if _, err := pool.Exec(ctx, `TRUNCATE TABLE notifications, events`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

// TestWorkerProcessAndDeliverEndToEnd tests the full worker pipeline:
// 1. Submit events via use case
// 2. ProcessEvents claims them and creates notifications
// 3. DeliverNotifications sends emails via fake sender
func TestWorkerProcessAndDeliverEndToEnd(t *testing.T) {
	pool, ctx := newTestPool(t)
	setupWorkerSchema(t, pool, ctx)

	eventRepo := postgres.EventRepository{Pool: pool}
	notifRepo := postgres.NotificationRepository{Pool: pool}
	renderer := email.NewTemplateRenderer()
	sender := &fakeEmailSender{}

	// Step 1: Submit events
	submitUC := usecases.EventService{Repo: eventRepo}
	payload := mustMarshal(t, entities.UserRegisteredPayload{
		UserID: "u1", Email: "alice@example.com", Name: "Alice",
	})
	_, err := submitUC.SubmitEvent(ctx, entities.EventTypeUserRegistered, payload, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	if err != nil {
		t.Fatalf("submit event: %v", err)
	}

	payload2 := mustMarshal(t, entities.OrderPaidPayload{
		OrderID: "o1", UserID: "u1", Amount: 49.99, Currency: "USD",
	})
	// OrderPaid needs email in payload for extractRecipient
	payload2WithEmail := []byte(`{"order_id":"o1","user_id":"u1","amount":49.99,"currency":"USD","email":"alice@example.com"}`)
	_, err = submitUC.SubmitEvent(ctx, entities.EventTypeOrderPaid, payload2WithEmail, "ffffffff-ffff-ffff-ffff-ffffffffffff")
	if err != nil {
		t.Fatalf("submit event 2: %v", err)
	}

	// Verify events are in 'accepted' status
	var acceptedCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM events WHERE status = 'accepted'`).Scan(&acceptedCount); err != nil {
		t.Fatalf("count accepted events: %v", err)
	}
	if acceptedCount != 2 {
		t.Fatalf("expected 2 accepted events, got %d", acceptedCount)
	}

	// Step 2: ProcessEvents
	processUC := usecases.ProcessEvents{
		EventRepo:        eventRepo,
		NotificationRepo: notifRepo,
		Renderer:         renderer,
		BatchSize:        10,
	}
	processed, err := processUC.Handle(ctx)
	if err != nil {
		t.Fatalf("process events: %v", err)
	}
	if processed != 2 {
		t.Fatalf("expected 2 processed, got %d", processed)
	}

	// Verify notifications were created with status 'pending'
	var notifCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE status = 'pending'`).Scan(&notifCount); err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if notifCount != 2 {
		t.Fatalf("expected 2 pending notifications, got %d", notifCount)
	}

	// Step 3: DeliverNotifications
	deliverUC := usecases.DeliverNotifications{
		NotificationRepo: notifRepo,
		Sender:           sender,
		BatchSize:        10,
	}
	delivered, err := deliverUC.Handle(ctx)
	if err != nil {
		t.Fatalf("deliver notifications: %v", err)
	}
	if delivered != 2 {
		t.Fatalf("expected 2 delivered, got %d", delivered)
	}

	// Verify emails were sent
	if len(sender.sent) != 2 {
		t.Fatalf("expected 2 emails sent, got %d", len(sender.sent))
	}
	for _, e := range sender.sent {
		if e.To != "alice@example.com" {
			t.Errorf("expected recipient alice@example.com, got %s", e.To)
		}
	}

	// Verify notifications are now 'delivered'
	var deliveredCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE status = 'delivered'`).Scan(&deliveredCount); err != nil {
		t.Fatalf("count delivered: %v", err)
	}
	if deliveredCount != 2 {
		t.Fatalf("expected 2 delivered notifications, got %d", deliveredCount)
	}

	// Suppress unused variable warning
	_ = payload2
}
