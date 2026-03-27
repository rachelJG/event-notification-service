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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// EventRepository is a concrete implementation of the ports.EventRepository
// interface for PostgreSQL. It uses the pgx library to interact with the
// database.
type EventRepository struct {
	//Pool is a connection pool to the PostgreSQL database
	Pool *pgxpool.Pool
	//QueryTimeout is the timeout for query operations, including the health check
	QueryTimeout time.Duration
}

// Create stores a new event in the database. It uses idempotency to avoid
// duplicates, allowing an explicit ID to be provided or generated
// automatically. If an event with the same idempotency key and type already
// exists, it updates the existing record.
func (r EventRepository) Create(ctx context.Context, event entities.Event) (string, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "EventRepository.Create",
		trace.WithAttributes(
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.table", "events"),
		),
	)
	defer span.End()

	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	id := event.ID
	if id == "" {
		id = uuid.NewString()
	}

	var clientID *string
	if event.ClientID != "" {
		clientID = &event.ClientID
	}

	err := r.Pool.QueryRow(ctx, `
		INSERT INTO events (id, type, idempotency_key, payload, notifications, client_id, occurred_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (idempotency_key, type)
		DO UPDATE SET id = events.id
		RETURNING id
	`, id, event.Type, event.IdempotencyKey, event.Payload, event.NotificationsJSON, clientID, event.OccurredAt, time.Now().UTC()).Scan(&id)
	if err != nil {
		span.SetStatus(codes.Error, "insert failed")
		span.RecordError(err)
		return "", mapDBError(err)
	}

	span.SetAttributes(attribute.String("event.id", id))
	return id, nil
}

func (r EventRepository) GetByID(ctx context.Context, id string) (entities.Event, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "EventRepository.GetByID",
		trace.WithAttributes(
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "events"),
			attribute.String("event.id", id),
		),
	)
	defer span.End()

	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	var e entities.Event
	var clientID *string
	err := r.Pool.QueryRow(ctx, `
		SELECT id, type, idempotency_key, payload, notifications, client_id, occurred_at, created_at
		FROM events
		WHERE id = $1
	`, id).Scan(&e.ID, &e.Type, &e.IdempotencyKey, &e.Payload, &e.NotificationsJSON, &clientID, &e.OccurredAt, &e.CreatedAt)
	if clientID != nil {
		e.ClientID = *clientID
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			span.SetStatus(codes.Error, "not found")
			return entities.Event{}, apperror.NotFound("event not found", err)
		}
		span.SetStatus(codes.Error, "query failed")
		span.RecordError(err)
		return entities.Event{}, mapDBError(err)
	}
	return e, nil
}

func (r EventRepository) ClaimPending(ctx context.Context, limit int) ([]entities.Event, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "EventRepository.ClaimPending",
		trace.WithAttributes(
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "events"),
			attribute.Int("db.limit", limit),
		),
	)
	defer span.End()

	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	rows, err := r.Pool.Query(ctx, `
		UPDATE events SET status = 'processing'
		WHERE id IN (
			SELECT id FROM events
			WHERE status = 'accepted'
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		RETURNING id, type, idempotency_key, payload, notifications, client_id, occurred_at, created_at
	`, limit)
	if err != nil {
		span.SetStatus(codes.Error, "query failed")
		span.RecordError(err)
		return nil, mapDBError(err)
	}
	defer rows.Close()

	var events []entities.Event
	for rows.Next() {
		var e entities.Event
		var clientID *string
		if err := rows.Scan(&e.ID, &e.Type, &e.IdempotencyKey, &e.Payload, &e.NotificationsJSON, &clientID, &e.OccurredAt, &e.CreatedAt); err != nil {
			span.SetStatus(codes.Error, "scan failed")
			span.RecordError(err)
			return nil, mapDBError(err)
		}
		if clientID != nil {
			e.ClientID = *clientID
		}
		events = append(events, e)
	}
	span.SetAttributes(attribute.Int("db.rows_returned", len(events)))
	return events, rows.Err()
}

func (r EventRepository) SetStatus(ctx context.Context, id string, status string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "EventRepository.SetStatus",
		trace.WithAttributes(
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "events"),
			attribute.String("event.id", id),
			attribute.String("event.status", status),
		),
	)
	defer span.End()

	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	_, err := r.Pool.Exec(ctx, `UPDATE events SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		span.SetStatus(codes.Error, "update failed")
		span.RecordError(err)
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
