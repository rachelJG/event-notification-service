package httpadapter

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rachelJG/event-notification-service/internal/adapters/http/dto"
	"github.com/rachelJG/event-notification-service/internal/core/usecases"
)

type Handler struct {
	SubmitEvent usecases.SubmitEvent
}

const idempotencyHeader = "Idempotency-Key"

func (h Handler) SubmitEventHandler(c *gin.Context) {
	if !isIdempotencyKeyValid(c.GetHeader(idempotencyHeader)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid Idempotency-Key"})
		return
	}

	var req dto.SubmitEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	id, err := h.SubmitEvent.Handle(c.Request.Context(), req.EventType, req.Payload, c.GetHeader(idempotencyHeader))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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
