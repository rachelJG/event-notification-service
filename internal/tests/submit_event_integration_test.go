//go:build integration

package tests

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/application/usecases"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/postgres"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

func newTestPool(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/events?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS events (
			id UUID PRIMARY KEY,
			type TEXT NOT NULL,
			idempotency_key TEXT NOT NULL,
			payload JSONB NOT NULL,
			occurred_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS ux_events_idempotency_type ON events (idempotency_key, type);
	`)
	if err != nil {
		t.Fatalf("ensure events table: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE TABLE events`); err != nil {
		t.Fatalf("truncate events: %v", err)
	}
	return pool, ctx
}

func newSubmitEventUseCase(pool *pgxpool.Pool) usecases.SubmitEvent {
	return usecases.SubmitEvent{Repo: postgres.EventRepository{Pool: pool}}
}

// TestSubmitEventAllEventTypes verifies that all supported event types can be
// submitted and persisted end-to-end through the use case.
func TestSubmitEventAllEventTypes(t *testing.T) {
	pool, ctx := newTestPool(t)
	uc := newSubmitEventUseCase(pool)

	cases := []struct {
		name           string
		eventType      string
		payload        []byte
		idempotencyKey string
	}{
		{
			name:           "UserRegistered",
			eventType:      entities.EventTypeUserRegistered,
			payload:        mustMarshal(t, entities.UserRegisteredPayload{UserID: "u1", Email: "u@test.com", Name: "Test"}),
			idempotencyKey: "11111111-1111-1111-1111-111111111111",
		},
		{
			name:           "PasswordResetRequested",
			eventType:      entities.EventTypePasswordResetRequested,
			payload:        mustMarshal(t, entities.PasswordResetRequestedPayload{UserID: "u1", Email: "u@test.com"}),
			idempotencyKey: "22222222-2222-2222-2222-222222222222",
		},
		{
			name:           "OrderPaid",
			eventType:      entities.EventTypeOrderPaid,
			payload:        mustMarshal(t, entities.OrderPaidPayload{OrderID: "o1", UserID: "u1", Amount: 99.99, Currency: "USD"}),
			idempotencyKey: "33333333-3333-3333-3333-333333333333",
		},
		{
			name:           "OrderShipped",
			eventType:      entities.EventTypeOrderShipped,
			payload:        mustMarshal(t, entities.OrderShippedPayload{OrderID: "o1", UserID: "u1", Carrier: "DHL", TrackingNumber: "TRK123"}),
			idempotencyKey: "44444444-4444-4444-4444-444444444444",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := uc.Handle(ctx, tc.eventType, tc.payload, tc.idempotencyKey)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if id == "" {
				t.Fatal("expected non-empty id")
			}
		})
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != len(cases) {
		t.Fatalf("expected %d events in DB, got %d", len(cases), count)
	}
}

// TestSubmitEventIdempotency verifies that submitting the same event twice
// (same idempotency key and type) returns the same ID and inserts only one row.
func TestSubmitEventIdempotency(t *testing.T) {
	pool, ctx := newTestPool(t)
	uc := newSubmitEventUseCase(pool)

	payload := mustMarshal(t, entities.UserRegisteredPayload{UserID: "u1", Email: "u@test.com", Name: "Test"})
	key := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	id1, err := uc.Handle(ctx, entities.EventTypeUserRegistered, payload, key)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}

	id2, err := uc.Handle(ctx, entities.EventTypeUserRegistered, payload, key)
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}

	if id1 != id2 {
		t.Fatalf("expected same id on duplicate submit, got %s and %s", id1, id2)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row in DB, got %d", count)
	}
}

// TestSubmitEventSameKeyDifferentType verifies that the same idempotency key
// used with different event types creates two independent events.
func TestSubmitEventSameKeyDifferentType(t *testing.T) {
	pool, ctx := newTestPool(t)
	uc := newSubmitEventUseCase(pool)

	key := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	id1, err := uc.Handle(ctx, entities.EventTypeOrderPaid,
		mustMarshal(t, entities.OrderPaidPayload{OrderID: "o1", UserID: "u1", Amount: 10, Currency: "USD"}),
		key,
	)
	if err != nil {
		t.Fatalf("OrderPaid submit: %v", err)
	}

	id2, err := uc.Handle(ctx, entities.EventTypeOrderShipped,
		mustMarshal(t, entities.OrderShippedPayload{OrderID: "o1", UserID: "u1", Carrier: "DHL", TrackingNumber: "TRK1"}),
		key,
	)
	if err != nil {
		t.Fatalf("OrderShipped submit: %v", err)
	}

	if id1 == id2 {
		t.Fatal("expected different ids for different event types with same idempotency key")
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows in DB, got %d", count)
	}
}

// TestSubmitEventInvalidPayloadNotPersisted verifies that a validation error
// prevents the event from being written to the database.
func TestSubmitEventInvalidPayloadNotPersisted(t *testing.T) {
	pool, ctx := newTestPool(t)
	uc := newSubmitEventUseCase(pool)

	// missing required user_id
	payload := []byte(`{"user_id":"","email":"u@test.com","name":"Test"}`)
	_, err := uc.Handle(ctx, entities.EventTypeUserRegistered, payload, "cccccccc-cccc-cccc-cccc-cccccccccccc")
	if err == nil {
		t.Fatal("expected validation error for invalid payload")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument error, got %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows in DB after failed submit, got %d", count)
	}
}

// TestSubmitEventUnsupportedTypeNotPersisted verifies that an unsupported event
// type is rejected before reaching the database.
func TestSubmitEventUnsupportedTypeNotPersisted(t *testing.T) {
	pool, ctx := newTestPool(t)
	uc := newSubmitEventUseCase(pool)

	_, err := uc.Handle(ctx, "UnknownEvent", []byte(`{}`), "dddddddd-dddd-dddd-dddd-dddddddddddd")
	if err == nil {
		t.Fatal("expected error for unsupported event type")
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows in DB, got %d", count)
	}
}

// TestSubmitEventEmptyIdempotencyKeyRejected verifies that a missing
// idempotency key is rejected without reaching the database.
func TestSubmitEventEmptyIdempotencyKeyRejected(t *testing.T) {
	pool, ctx := newTestPool(t)
	uc := newSubmitEventUseCase(pool)

	payload := mustMarshal(t, entities.UserRegisteredPayload{UserID: "u1", Email: "u@test.com", Name: "Test"})
	_, err := uc.Handle(ctx, entities.EventTypeUserRegistered, payload, "")
	if err == nil {
		t.Fatal("expected error for empty idempotency key")
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows in DB, got %d", count)
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return b
}
