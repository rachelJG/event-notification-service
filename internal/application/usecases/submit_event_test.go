package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
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

func TestSubmitEventHandleSuccess(t *testing.T) {
	repo := &fakeRepo{}
	uc := SubmitEvent{Repo: repo}

	id, err := uc.Handle(context.Background(), entities.EventTypeUserRegistered, []byte(`{"user_id":"1","email":"a@b.com","name":"A"}`), "idem-1")
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
}

func TestSubmitEventHandleValidationError(t *testing.T) {
	repo := &fakeRepo{}
	uc := SubmitEvent{Repo: repo}

	_, err := uc.Handle(context.Background(), entities.EventTypeUserRegistered, []byte(`{"user_id":"","email":"a@b.com","name":"A"}`), "idem-1")
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestSubmitEventHandleRepoError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("db error")}
	uc := SubmitEvent{Repo: repo}

	_, err := uc.Handle(context.Background(), entities.EventTypeUserRegistered, []byte(`{"user_id":"1","email":"a@b.com","name":"A"}`), "idem-1")
	if err == nil {
		t.Fatalf("expected repo error")
	}
}

func TestSubmitEventHandleMissingIdempotencyKey(t *testing.T) {
	repo := &fakeRepo{}
	uc := SubmitEvent{Repo: repo}

	_, err := uc.Handle(context.Background(), entities.EventTypeUserRegistered, []byte(`{"user_id":"1","email":"a@b.com","name":"A"}`), "")
	if err == nil {
		t.Fatalf("expected idempotency key error")
	}
}

func TestSubmitEventHandleEmptyEventType(t *testing.T) {
	repo := &fakeRepo{}
	uc := SubmitEvent{Repo: repo}

	_, err := uc.Handle(context.Background(), "", []byte(`{"user_id":"1","email":"a@b.com","name":"A"}`), "idem-1")
	if err == nil {
		t.Fatal("expected error for empty event_type")
	}
}

func TestSubmitEventHandleUnsupportedEventType(t *testing.T) {
	repo := &fakeRepo{}
	uc := SubmitEvent{Repo: repo}

	_, err := uc.Handle(context.Background(), "UnknownType", []byte(`{"user_id":"1"}`), "idem-1")
	if err == nil {
		t.Fatal("expected error for unsupported event_type")
	}
}
