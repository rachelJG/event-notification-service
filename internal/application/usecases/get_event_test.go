package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

func TestGetEventHandleSuccess(t *testing.T) {
	want := entities.Event{ID: "evt-1", Type: entities.EventTypeOrderPaid}
	repo := &fakeRepo{storedByID: map[string]entities.Event{"evt-1": want}}
	uc := GetEvent{Repo: repo}

	got, err := uc.Handle(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != want.ID {
		t.Fatalf("expected id %s, got %s", want.ID, got.ID)
	}
}

func TestGetEventHandleEmptyID(t *testing.T) {
	repo := &fakeRepo{}
	uc := GetEvent{Repo: repo}

	_, err := uc.Handle(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty id")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument error, got %v", err)
	}
}

func TestGetEventHandleRepoError(t *testing.T) {
	repo := &fakeRepo{err: apperror.NotFound("event not found", nil)}
	uc := GetEvent{Repo: repo}

	_, err := uc.Handle(context.Background(), "non-existent")
	if err == nil {
		t.Fatal("expected error from repo")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeNotFound {
		t.Fatalf("expected not_found error, got %v", err)
	}
}
