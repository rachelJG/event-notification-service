package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

type EventRepository struct {
	Pool         *pgxpool.Pool
	QueryTimeout time.Duration
}

func (r EventRepository) Create(ctx context.Context, event entities.Event) (string, error) {
	if r.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.QueryTimeout)
		defer cancel()
	}

	id := event.ID
	if id == "" {
		id = uuid.NewString()
	}

	err := r.Pool.QueryRow(ctx, `
		INSERT INTO events (id, type, idempotency_key, payload, occurred_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (idempotency_key, type)
		DO UPDATE SET id = events.id
		RETURNING id
	`, id, event.Type, event.IdempotencyKey, event.Payload, event.OccurredAt, time.Now().UTC()).Scan(&id)
	if err != nil {
		return "", err
	}

	return id, nil
}
