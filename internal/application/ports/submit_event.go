package ports

import "context"

// Input port: use cases exposed to driving adapters (HTTP, gRPC, etc.).
type SubmitEventUseCase interface {
	Handle(ctx context.Context, eventType string, payload []byte, idempotencyKey string) (string, error)
}
