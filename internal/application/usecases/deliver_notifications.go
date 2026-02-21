package usecases

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
)

type DeliverNotifications struct {
	NotificationRepo ports.NotificationRepository
	Sender           ports.EmailSender
	BatchSize        int
}

func (uc DeliverNotifications) Handle(ctx context.Context) (int, error) {
	notifications, err := uc.NotificationRepo.FindPending(ctx, uc.BatchSize)
	if err != nil {
		return 0, fmt.Errorf("find pending notifications: %w", err)
	}

	delivered := 0
	for _, n := range notifications {
		if err := uc.NotificationRepo.UpdateStatus(ctx, n.ID, ports.NotificationUpdate{
			Status:   entities.NotificationStatusProcessing,
			Attempts: n.Attempts,
		}); err != nil {
			continue
		}

		if err := uc.Sender.Send(ctx, n.Recipient, n.Subject, n.Body); err != nil {
			n.Attempts++
			lastError := err.Error()
			if n.Attempts >= n.MaxAttempts {
				_ = uc.NotificationRepo.UpdateStatus(ctx, n.ID, ports.NotificationUpdate{
					Status:    entities.NotificationStatusFailed,
					Attempts:  n.Attempts,
					LastError: lastError,
				})
			} else {
				_ = uc.NotificationRepo.UpdateStatus(ctx, n.ID, ports.NotificationUpdate{
					Status:      entities.NotificationStatusPending,
					Attempts:    n.Attempts,
					LastError:   lastError,
					NextRetryAt: time.Now().Add(retryDelay(n.Attempts)),
				})
			}
			continue
		}

		_ = uc.NotificationRepo.UpdateStatus(ctx, n.ID, ports.NotificationUpdate{
			Status:   entities.NotificationStatusDelivered,
			Attempts: n.Attempts,
		})
		delivered++
	}

	return delivered, nil
}

// retryDelay calculates exponential backoff delay for a given attempt number.
func retryDelay(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}
