package httpadapter

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rachelJG/event-notification-service/internal/core/usecases"
)

type Handler struct {
	SubmitEvent usecases.SubmitEvent
}

type submitEventRequest struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

type submitEventResponse struct {
	ID string `json:"id"`
}

func (h Handler) SubmitEventHandler(c *gin.Context) {
	var req submitEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	id, err := h.SubmitEvent.Handle(c.Request.Context(), req.EventType, req.Payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, submitEventResponse{ID: id})
}
