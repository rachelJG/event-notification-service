package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
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
		return "", mapDBError(err)
	}

	return id, nil
}

// mapDBError translates pgx/pgconn errors into AppError codes so that the
// application layer never needs to import database-specific packages.
func mapDBError(err error) error {
	// Let context errors pass through — errmap handles them at the HTTP layer.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return apperror.Conflict("duplicate event", err)
		case "23503": // foreign_key_violation
			return apperror.Conflict("referenced entity not found", err)
		case "23502": // not_null_violation
			return apperror.InvalidArgument("missing required field", err)
		}
	}

	return apperror.Internal("database error", err)
}
