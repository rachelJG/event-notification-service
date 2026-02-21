package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

func TestMapDBErrorContextDeadline(t *testing.T) {
	err := mapDBError(context.DeadlineExceeded)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded to pass through, got %v", err)
	}
}

func TestMapDBErrorContextCanceled(t *testing.T) {
	err := mapDBError(context.Canceled)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled to pass through, got %v", err)
	}
}

func TestMapDBErrorUniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505"}
	err := mapDBError(pgErr)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperror.CodeConflict {
		t.Fatalf("expected conflict code, got %s", appErr.Code)
	}
}

func TestMapDBErrorForeignKeyViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23503"}
	err := mapDBError(pgErr)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperror.CodeConflict {
		t.Fatalf("expected conflict code, got %s", appErr.Code)
	}
}

func TestMapDBErrorNotNullViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23502"}
	err := mapDBError(pgErr)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperror.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument code, got %s", appErr.Code)
	}
}

func TestMapDBErrorUnknown(t *testing.T) {
	err := mapDBError(errors.New("some unknown db error"))

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != apperror.CodeInternal {
		t.Fatalf("expected internal code, got %s", appErr.Code)
	}
}
