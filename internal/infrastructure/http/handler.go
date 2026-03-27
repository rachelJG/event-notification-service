package httpadapter

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appports "github.com/rachelJG/event-notification-service/internal/application/ports"
	"github.com/rachelJG/event-notification-service/internal/application/validation"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/dto"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/errmap"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

type Handler struct {
	EventService appports.EventService
}

const idempotencyHeader = "Idempotency-Key"

func (h Handler) SubmitEventHandler(c *gin.Context) {
	if !isIdempotencyKeyValid(c.GetHeader(idempotencyHeader)) {
		_ = c.Error(apperror.InvalidArgument("missing or invalid Idempotency-Key", nil))
		return
	}

	var req dto.SubmitEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.InvalidArgument("invalid JSON body", err))
		return
	}
	c.Set("event_type", req.EventType)
	c.Set("idempotency_key", c.GetHeader(idempotencyHeader))

	clientID := c.GetString("client_id")

	notifications := make([]validation.NotificationSpec, len(req.Notifications))
	for i, n := range req.Notifications {
		notifications[i] = validation.NotificationSpec{
			Channel:    n.Channel,
			From:       n.From,
			Subject:    n.Subject,
			Body:       n.Body,
			Recipients: n.Recipients,
		}
	}

	id, err := h.EventService.SubmitEvent(c.Request.Context(), req.EventType, req.Payload, notifications, c.GetHeader(idempotencyHeader), clientID)
	if err != nil {
		recordEventSubmitted(req.EventType, "error")
		httpErr := errmap.FromError(err)
		recordHTTPError(httpErr.Code)
		_ = c.Error(err)
		return
	}

	recordEventSubmitted(req.EventType, "success")

	c.Header("Location", "/api/v1/events/"+id)
	c.JSON(http.StatusAccepted, dto.SubmitEventResponse{ID: id})
}

func isIdempotencyKeyValid(key string) bool {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return false
	}
	_, err := uuid.Parse(trimmed)
	return err == nil
}

func (h Handler) GetEventHandler(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		_ = c.Error(apperror.InvalidArgument("invalid event ID format", err))
		return
	}
	event, err := h.EventService.GetEvent(c.Request.Context(), id)
	if err != nil {
		httpErr := errmap.FromError(err)
		recordHTTPError(httpErr.Code)
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.GetEventResponse{
		ID:         event.ID,
		Type:       event.Type,
		Payload:    event.Payload,
		ClientID:   event.ClientID,
		OccurredAt: event.OccurredAt,
		CreatedAt:  event.CreatedAt,
	})
}
