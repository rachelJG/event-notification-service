package httpadapter

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appports "github.com/rachelJG/event-notification-service/internal/application/ports"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/dto"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/errmap"
	"go.uber.org/zap"
)

type Handler struct {
	SubmitEvent appports.SubmitEventUseCase
	GetEvent    appports.GetEventUseCase
	Logger      *zap.Logger
}

const idempotencyHeader = "Idempotency-Key"

func (h Handler) SubmitEventHandler(c *gin.Context) {
	if !isIdempotencyKeyValid(c.GetHeader(idempotencyHeader)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid Idempotency-Key", "code": "invalid_argument"})
		return
	}

	var req dto.SubmitEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body", "code": "invalid_argument"})
		return
	}
	c.Set("event_type", req.EventType)
	c.Set("idempotency_key", c.GetHeader(idempotencyHeader))

	id, err := h.SubmitEvent.Handle(c.Request.Context(), req.EventType, req.Payload, c.GetHeader(idempotencyHeader))
	if err != nil {
		recordEventSubmitted(req.EventType, "error")
		h.logError(c, err)
		httpErr := errmap.FromError(err)
		c.JSON(httpErr.Status, gin.H{"error": errmap.Message(err), "code": httpErr.Code})
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
		zap.String("request_id", requestIDFromContext(c)),
		zap.String("event_type", eventTypeFromContext(c)),
		zap.String("idempotency_key", idempotencyKeyFromContext(c)),
		zap.Error(err),
	)
}

func requestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func eventTypeFromContext(c *gin.Context) string {
	if v, ok := c.Get("event_type"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func idempotencyKeyFromContext(c *gin.Context) string {
	if v, ok := c.Get("idempotency_key"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (h Handler) GetEventHandler(c *gin.Context) {
	id := c.Param("id")
	event, err := h.GetEvent.Handle(c.Request.Context(), id)
	if err != nil {
		h.logError(c, err)
		httpErr := errmap.FromError(err)
		c.JSON(httpErr.Status, gin.H{"error": errmap.Message(err), "code": httpErr.Code})
		return
	}
	c.JSON(http.StatusOK, dto.GetEventResponse{
		ID:         event.ID,
		Type:       event.Type,
		Payload:    event.Payload,
		OccurredAt: event.OccurredAt,
		CreatedAt:  event.CreatedAt,
	})
}
