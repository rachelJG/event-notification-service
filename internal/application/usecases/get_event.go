package usecases

import (
	"context"
	"strings"

	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
)

type GetEvent struct {
	Repo ports.EventRepository
}

func (uc GetEvent) Handle(ctx context.Context, id string) (entities.Event, error) {
	if strings.TrimSpace(id) == "" {
		return entities.Event{}, apperror.InvalidArgument("id is required", nil)
	}
	return uc.Repo.GetByID(ctx, id)
}
