package httpadapter

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rachelJG/event-notification-service/internal/adapters/http/dto"
	"github.com/rachelJG/event-notification-service/internal/core/apperror"
	"github.com/rachelJG/event-notification-service/internal/core/usecases"
	"go.uber.org/zap"
)

type Handler struct {
	SubmitEvent usecases.SubmitEvent
	Logger      *zap.Logger
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
		h.logError(c, err)
		status, code := httpStatusFromError(err)
		c.JSON(status, gin.H{"error": errorMessage(err), "code": code})
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

func httpStatusFromError(err error) (int, string) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case apperror.CodeInvalidArgument:
			return http.StatusBadRequest, string(appErr.Code)
		case apperror.CodeNotFound:
			return http.StatusNotFound, string(appErr.Code)
		case apperror.CodeConflict:
			return http.StatusConflict, string(appErr.Code)
		default:
			return http.StatusInternalServerError, string(apperror.CodeInternal)
		}
	}
	return http.StatusInternalServerError, string(apperror.CodeInternal)
}

func errorMessage(err error) string {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) && appErr.Message != "" {
		return appErr.Message
	}
	return "internal error"
}

func (h Handler) logError(c *gin.Context, err error) {
	if h.Logger == nil {
		return
	}
	status, code := httpStatusFromError(err)
	h.Logger.Error("request failed",
		zap.String("method", c.Request.Method),
		zap.String("path", c.FullPath()),
		zap.Int("status", status),
		zap.String("code", code),
		zap.Error(err),
	)
}
