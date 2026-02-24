//go:build integration

package tests

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/postgres"
)

func TestPostgresEventRepositoryCreate(t *testing.T) {
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/events?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

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

	// TRUNCATE CASCADE handles foreign key constraints from notifications table
	_, err = pool.Exec(ctx, `TRUNCATE TABLE events CASCADE`)
	if err != nil {
		t.Fatalf("truncate events: %v", err)
	}

	repo := postgres.EventRepository{Pool: pool}
	payload, _ := json.Marshal(entities.UserRegisteredPayload{
		UserID: "1",
		Email:  "test@example.com",
		Name:   "Test User",
	})

	id, err := repo.Create(ctx, entities.Event{
		Type:           entities.EventTypeUserRegistered,
		IdempotencyKey: "idem-1",
		Payload:        payload,
		OccurredAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	if id == "" {
		t.Fatalf("expected id to be generated")
	}

	id2, err := repo.Create(ctx, entities.Event{
		Type:           entities.EventTypeUserRegistered,
		IdempotencyKey: "idem-1",
		Payload:        payload,
		OccurredAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create event with same idempotency key: %v", err)
	}
	if id2 != id {
		t.Fatalf("expected idempotent insert to return same id, got %s", id2)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 event, got %d", count)
	}
}
