package httpadapter

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appports "github.com/rachelJG/event-notification-service/internal/application/ports"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/dto"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/errmap"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/httputil"
	"go.uber.org/zap"
)

type Handler struct {
	EventService appports.EventService
	Logger       *zap.Logger
}

const idempotencyHeader = "Idempotency-Key"

func (h Handler) SubmitEventHandler(c *gin.Context) {
	if !isIdempotencyKeyValid(c.GetHeader(idempotencyHeader)) {
		httputil.WriteCustomError(c, http.StatusBadRequest, "missing or invalid Idempotency-Key", "invalid_argument")
		return
	}

	var req dto.SubmitEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.WriteCustomError(c, http.StatusBadRequest, "invalid JSON body", "invalid_argument")
		return
	}
	c.Set("event_type", req.EventType)
	c.Set("idempotency_key", c.GetHeader(idempotencyHeader))

	// Get client_id from authenticated API key context (may be empty for backward compatibility)
	clientID := c.GetString("client_id")

	id, err := h.EventService.SubmitEvent(c.Request.Context(), req.EventType, req.Payload, c.GetHeader(idempotencyHeader), clientID)
	if err != nil {
		recordEventSubmitted(req.EventType, "error")
		h.logError(c, err)
		httpErr := errmap.FromError(err)
		recordHTTPError(httpErr.Code)
		httputil.WriteCustomError(c, httpErr.Status, errmap.Message(err), httpErr.Code)
		return
	}

	recordEventSubmitted(req.EventType, "success")

	// Set the Location header to the URL of the newly created event.
	// This is a standard HTTP response header used to indicate the URL of the
	// resource that has been created. In this case, it specifies the URL of the
	// newly created event. The client can then use this URL to retrieve the
	// event if needed.
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

func (h Handler) logError(c *gin.Context, err error) {
	if h.Logger == nil {
		return
	}
	httpErr := errmap.FromError(err)
	h.Logger.Error("request failed",
		zap.String("method", c.Request.Method),
		zap.String("path", c.FullPath()),
		zap.Int("status", httpErr.Status),
		zap.String("code", httpErr.Code),
		zap.String("request_id", c.GetString("request_id")),
		zap.String("event_type", c.GetString("event_type")),
		zap.String("idempotency_key", c.GetString("idempotency_key")),
		zap.Error(err),
	)
}


func (h Handler) GetEventHandler(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httputil.WriteCustomError(c, http.StatusBadRequest, "invalid event ID format", "invalid_argument")
		return
	}
	event, err := h.EventService.GetEvent(c.Request.Context(), id)
	if err != nil {
		h.logError(c, err)
		httpErr := errmap.FromError(err)
		recordHTTPError(httpErr.Code)
		httputil.WriteCustomError(c, httpErr.Status, errmap.Message(err), httpErr.Code)
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
