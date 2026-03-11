package ports

import (
	"context"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
)

// APIKeyRepository is an output port for managing API key persistence.
type APIKeyRepository interface {
	Create(ctx context.Context, key entities.APIKey) error
	GetByHash(ctx context.Context, keyHash string) (entities.APIKey, error)
	List(ctx context.Context) ([]entities.APIKey, error)
	Revoke(ctx context.Context, id string) error
	UpdateLastUsed(ctx context.Context, id string) error
}
