package ports

import "context"

// Input port for delivering pending notifications via their channel.
type DeliverNotificationsUseCase interface {
	Handle(ctx context.Context) (int, error)
}
