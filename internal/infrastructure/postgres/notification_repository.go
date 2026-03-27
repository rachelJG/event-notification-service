package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type NotificationRepository struct {
	Pool         *pgxpool.Pool
	QueryTimeout time.Duration
}

func (r NotificationRepository) Create(ctx context.Context, n entities.Notification) (string, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "NotificationRepository.Create",
		trace.WithAttributes(
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.table", "notifications"),
			attribute.String("notification.channel", string(n.Channel)),
		),
	)
	defer span.End()

	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	id := n.ID
	if id == "" {
		id = uuid.NewString()
	}

	err := r.Pool.QueryRow(ctx, `
		INSERT INTO notifications (id, event_id, channel, recipient, subject, body, status, attempts, max_attempts, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`, id, n.EventID, n.Channel, n.Recipient, n.Subject, n.Body, n.Status, n.Attempts, n.MaxAttempts, n.CreatedAt, n.UpdatedAt).Scan(&id)
	if err != nil {
		span.SetStatus(codes.Error, "insert failed")
		span.RecordError(err)
		return "", mapDBError(err)
	}

	return id, nil
}

func (r NotificationRepository) FindPending(ctx context.Context, limit int) ([]entities.Notification, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "NotificationRepository.FindPending",
		trace.WithAttributes(
			attribute.String("db.operation", "SELECT"),
			attribute.String("db.table", "notifications"),
			attribute.Int("db.limit", limit),
		),
	)
	defer span.End()

	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	rows, err := r.Pool.Query(ctx, `
		SELECT id, event_id, channel, recipient, subject, body, status, attempts, max_attempts,
		       last_error, next_retry_at, created_at, updated_at
		FROM notifications
		WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at
		LIMIT $1
	`, limit)
	if err != nil {
		span.SetStatus(codes.Error, "query failed")
		span.RecordError(err)
		return nil, mapDBError(err)
	}
	defer rows.Close()

	var notifications []entities.Notification
	for rows.Next() {
		var n entities.Notification
		var lastError *string
		var nextRetryAt *time.Time
		if err := rows.Scan(
			&n.ID, &n.EventID, &n.Channel, &n.Recipient, &n.Subject, &n.Body,
			&n.Status, &n.Attempts, &n.MaxAttempts,
			&lastError, &nextRetryAt, &n.CreatedAt, &n.UpdatedAt,
		); err != nil {
			span.SetStatus(codes.Error, "scan failed")
			span.RecordError(err)
			return nil, mapDBError(err)
		}
		if lastError != nil {
			n.LastError = *lastError
		}
		if nextRetryAt != nil {
			n.NextRetryAt = *nextRetryAt
		}
		notifications = append(notifications, n)
	}
	span.SetAttributes(attribute.Int("db.rows_returned", len(notifications)))
	return notifications, rows.Err()
}

func (r NotificationRepository) UpdateStatus(ctx context.Context, id string, update ports.NotificationUpdate) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "NotificationRepository.UpdateStatus",
		trace.WithAttributes(
			attribute.String("db.operation", "UPDATE"),
			attribute.String("db.table", "notifications"),
			attribute.String("notification.id", id),
			attribute.String("notification.status", string(update.Status)),
		),
	)
	defer span.End()

	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	var nextRetryAt *time.Time
	if !update.NextRetryAt.IsZero() {
		nextRetryAt = &update.NextRetryAt
	}

	var lastError *string
	if update.LastError != "" {
		lastError = &update.LastError
	}

	ct, err := r.Pool.Exec(ctx, `
		UPDATE notifications
		SET status = $1, attempts = $2, last_error = $3, next_retry_at = $4, updated_at = NOW()
		WHERE id = $5
	`, update.Status, update.Attempts, lastError, nextRetryAt, id)
	if err != nil {
		span.SetStatus(codes.Error, "update failed")
		span.RecordError(err)
		return mapDBError(err)
	}
	if ct.RowsAffected() == 0 {
		span.SetStatus(codes.Error, "not found")
		return apperror.NotFound("notification not found", errors.New("no rows affected"))
	}
	return nil
}

// Ensure NotificationRepository implements ports.NotificationRepository.
var _ ports.NotificationRepository = NotificationRepository{}
