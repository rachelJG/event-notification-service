package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

type EventRepository struct {
	Pool         *pgxpool.Pool
	QueryTimeout time.Duration
}

// Create stores a new event in the database. It uses idempotency to avoid
// duplicates, allowing an explicit ID to be provided or generated
// automatically. If an event with the same idempotency key and type already
// exists, it updates the existing record.
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

	var clientID *string
	if event.ClientID != "" {
		clientID = &event.ClientID
	}

	err := r.Pool.QueryRow(ctx, `
		INSERT INTO events (id, type, idempotency_key, payload, client_id, occurred_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (idempotency_key, type)
		DO UPDATE SET id = events.id
		RETURNING id
	`, id, event.Type, event.IdempotencyKey, event.Payload, clientID, event.OccurredAt, time.Now().UTC()).Scan(&id)
	if err != nil {
		return "", mapDBError(err)
	}

	return id, nil
}

func (r EventRepository) GetByID(ctx context.Context, id string) (entities.Event, error) {
	if r.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.QueryTimeout)
		defer cancel()
	}

	var e entities.Event
	var clientID *string
	err := r.Pool.QueryRow(ctx, `
		SELECT id, type, idempotency_key, payload, client_id, occurred_at, created_at
		FROM events
		WHERE id = $1
	`, id).Scan(&e.ID, &e.Type, &e.IdempotencyKey, &e.Payload, &clientID, &e.OccurredAt, &e.CreatedAt)
	if clientID != nil {
		e.ClientID = *clientID
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.Event{}, apperror.NotFound("event not found", err)
		}
		return entities.Event{}, mapDBError(err)
	}
	return e, nil
}

func (r EventRepository) ClaimPending(ctx context.Context, limit int) ([]entities.Event, error) {
	if r.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.QueryTimeout)
		defer cancel()
	}

	rows, err := r.Pool.Query(ctx, `
		UPDATE events SET status = 'processing'
		WHERE id IN (
			SELECT id FROM events
			WHERE status = 'accepted'
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		RETURNING id, type, idempotency_key, payload, client_id, occurred_at, created_at
	`, limit)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	var events []entities.Event
	for rows.Next() {
		var e entities.Event
		var clientID *string
		if err := rows.Scan(&e.ID, &e.Type, &e.IdempotencyKey, &e.Payload, &clientID, &e.OccurredAt, &e.CreatedAt); err != nil {
			return nil, mapDBError(err)
		}
		if clientID != nil {
			e.ClientID = *clientID
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r EventRepository) SetStatus(ctx context.Context, id string, status string) error {
	if r.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.QueryTimeout)
		defer cancel()
	}

	_, err := r.Pool.Exec(ctx, `UPDATE events SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return mapDBError(err)
	}
	return nil
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
