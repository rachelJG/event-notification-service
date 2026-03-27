package usecases

import (
	"context"
	"math"
	"time"

	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/domain/ports"
	"github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type DeliverNotifications struct {
	NotificationRepo ports.NotificationRepository
	Sender           ports.EmailSender
	WhatsAppSender   ports.WhatsAppSender
	BatchSize        int
}

func (uc DeliverNotifications) Handle(ctx context.Context) (int, error) {
	ctx, span := otel.Tracer("usecases").Start(ctx, "DeliverNotifications.Handle",
		trace.WithAttributes(attribute.Int("batch_size", uc.BatchSize)),
	)
	defer span.End()

	notifications, err := uc.NotificationRepo.FindPending(ctx, uc.BatchSize)
	if err != nil {
		span.SetStatus(codes.Error, "find pending failed")
		span.RecordError(err)
		return 0, apperror.Internal("find pending notifications", err)
	}

	span.SetAttributes(attribute.Int("notifications.pending", len(notifications)))

	delivered := 0
	for _, n := range notifications {
		if err := uc.NotificationRepo.UpdateStatus(ctx, n.ID, ports.NotificationUpdate{
			Status:   entities.NotificationStatusProcessing,
			Attempts: n.Attempts,
		}); err != nil {
			continue
		}

		if err := uc.send(ctx, n); err != nil {
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

	span.SetAttributes(attribute.Int("notifications.delivered", delivered))
	return delivered, nil
}

// send dispatches the notification to the appropriate channel sender.
func (uc DeliverNotifications) send(ctx context.Context, n entities.Notification) error {
	ctx, span := otel.Tracer("usecases").Start(ctx, "DeliverNotifications.send",
		trace.WithAttributes(
			attribute.String("notification.id", n.ID),
			attribute.String("notification.channel", string(n.Channel)),
			attribute.Int("notification.attempt", n.Attempts),
		),
	)
	defer span.End()

	var err error
	switch n.Channel {
	case entities.ChannelEmail:
		err = uc.Sender.Send(ctx, n.From, n.Recipient, n.Subject, n.Body)
	case entities.ChannelWhatsApp:
		if uc.WhatsAppSender == nil {
			err = apperror.Internal("whatsapp sender not configured", nil)
		} else {
			err = uc.WhatsAppSender.SendToGroup(ctx, n.Recipient, n.Body)
		}
	default:
		err = apperror.InvalidArgument("unsupported channel: "+string(n.Channel), nil)
	}

	if err != nil {
		span.SetStatus(codes.Error, "send failed")
		span.RecordError(err)
	}
	return err
}

// retryDelay calculates exponential backoff delay for a given attempt number.
func retryDelay(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}
