package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

type fakeRepo struct {
	lastEvent  entities.Event
	storedByID map[string]entities.Event
	err        error
}

func (r *fakeRepo) Create(ctx context.Context, event entities.Event) (string, error) {
	r.lastEvent = event
	if r.err != nil {
		return "", r.err
	}
	return "evt-123", nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id string) (entities.Event, error) {
	if r.err != nil {
		return entities.Event{}, r.err
	}
	if r.storedByID != nil {
		if e, ok := r.storedByID[id]; ok {
			return e, nil
		}
	}
	return entities.Event{}, nil
}

func (r *fakeRepo) ClaimPending(ctx context.Context, limit int) ([]entities.Event, error) {
	return nil, nil
}

func (r *fakeRepo) SetStatus(ctx context.Context, id string, status string) error {
	return nil
}

// SubmitEvent tests

func TestSubmitEventSuccess(t *testing.T) {
	repo := &fakeRepo{}
	svc := EventService{Repo: repo}

	id, err := svc.SubmitEvent(context.Background(), entities.EventTypeUserRegistered, []byte(`{"user_id":"1","email":"a@b.com","name":"A"}`), "idem-1", "test-client")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "evt-123" {
		t.Fatalf("expected id to match, got %s", id)
	}
	if repo.lastEvent.Type != entities.EventTypeUserRegistered {
		t.Fatalf("expected event type to be stored")
	}
	if repo.lastEvent.IdempotencyKey != "idem-1" {
		t.Fatalf("expected idempotency key to be stored")
	}
	if repo.lastEvent.ClientID != "test-client" {
		t.Fatalf("expected client_id to be stored, got %s", repo.lastEvent.ClientID)
	}
}

func TestSubmitEventValidationError(t *testing.T) {
	repo := &fakeRepo{}
	svc := EventService{Repo: repo}

	_, err := svc.SubmitEvent(context.Background(), entities.EventTypeUserRegistered, []byte(`{"user_id":"","email":"a@b.com","name":"A"}`), "idem-1", "")
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestSubmitEventRepoError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("db error")}
	svc := EventService{Repo: repo}

	_, err := svc.SubmitEvent(context.Background(), entities.EventTypeUserRegistered, []byte(`{"user_id":"1","email":"a@b.com","name":"A"}`), "idem-1", "")
	if err == nil {
		t.Fatalf("expected repo error")
	}
}

func TestSubmitEventMissingIdempotencyKey(t *testing.T) {
	repo := &fakeRepo{}
	svc := EventService{Repo: repo}

	_, err := svc.SubmitEvent(context.Background(), entities.EventTypeUserRegistered, []byte(`{"user_id":"1","email":"a@b.com","name":"A"}`), "", "")
	if err == nil {
		t.Fatalf("expected idempotency key error")
	}
}

func TestSubmitEventEmptyEventType(t *testing.T) {
	repo := &fakeRepo{}
	svc := EventService{Repo: repo}

	_, err := svc.SubmitEvent(context.Background(), "", []byte(`{"user_id":"1","email":"a@b.com","name":"A"}`), "idem-1", "")
	if err == nil {
		t.Fatal("expected error for empty event_type")
	}
}

func TestSubmitEventUnsupportedEventType(t *testing.T) {
	repo := &fakeRepo{}
	svc := EventService{Repo: repo}

	_, err := svc.SubmitEvent(context.Background(), "UnknownType", []byte(`{"user_id":"1"}`), "idem-1", "")
	if err == nil {
		t.Fatal("expected error for unsupported event_type")
	}
}

// GetEvent tests

func TestGetEventSuccess(t *testing.T) {
	want := entities.Event{ID: "evt-1", Type: entities.EventTypeOrderPaid}
	repo := &fakeRepo{storedByID: map[string]entities.Event{"evt-1": want}}
	svc := EventService{Repo: repo}

	got, err := svc.GetEvent(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != want.ID {
		t.Fatalf("expected id %s, got %s", want.ID, got.ID)
	}
}

func TestGetEventEmptyID(t *testing.T) {
	repo := &fakeRepo{}
	svc := EventService{Repo: repo}

	_, err := svc.GetEvent(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty id")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument error, got %v", err)
	}
}

func TestGetEventRepoError(t *testing.T) {
	repo := &fakeRepo{err: apperror.NotFound("event not found", nil)}
	svc := EventService{Repo: repo}

	_, err := svc.GetEvent(context.Background(), "non-existent")
	if err == nil {
		t.Fatal("expected error from repo")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeNotFound {
		t.Fatalf("expected not_found error, got %v", err)
	}
}
