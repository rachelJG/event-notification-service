package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/core/domain"
)

type EventRepository struct {
	Pool *pgxpool.Pool
}

func (r EventRepository) Create(ctx context.Context, event domain.Event) (string, error) {
	id := event.ID
	if id == "" {
		id = uuid.NewString()
	}

	_, err := r.Pool.Exec(ctx, `
		INSERT INTO events (id, type, payload, occurred_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, id, event.Type, event.Payload, event.OccurredAt, time.Now().UTC())
	if err != nil {
		return "", err
	}

	return id, nil
}
