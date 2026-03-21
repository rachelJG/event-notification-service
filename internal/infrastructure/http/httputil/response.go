package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

// ErrorResponse represents a standard JSON error response structure.
type ErrorResponse struct {
	Error     string `json:"error"`
	Code      string `json:"code"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteError writes a standardized JSON error response to the client and aborts
// the request. It maps AppError codes to appropriate HTTP status codes.
func WriteError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	code := string(apperror.CodeInternal)
	message := "internal error"

	if appErr, ok := err.(*apperror.AppError); ok {
		switch appErr.Code {
		case apperror.CodeUnauthenticated:
			status = http.StatusUnauthorized
			code = string(appErr.Code)
			message = appErr.Message
		case apperror.CodePermissionDenied:
			status = http.StatusForbidden
			code = string(appErr.Code)
			message = appErr.Message
		case apperror.CodeInternal:
			status = http.StatusInternalServerError
			code = string(appErr.Code)
			message = appErr.Message
		case apperror.CodeInvalidArgument:
			status = http.StatusBadRequest
			code = string(appErr.Code)
			message = appErr.Message
		case apperror.CodeNotFound:
			status = http.StatusNotFound
			code = string(appErr.Code)
			message = appErr.Message
		case apperror.CodeConflict:
			status = http.StatusConflict
			code = string(appErr.Code)
			message = appErr.Message
		case apperror.CodeRateLimited:
			status = http.StatusTooManyRequests
			code = string(appErr.Code)
			message = appErr.Message
		}
	}

	c.AbortWithStatusJSON(status, ErrorResponse{
		Error:     message,
		Code:      code,
		RequestID: c.GetString("request_id"),
	})
}

// WritePermissionError writes a 403 Forbidden error with a specific scope message.
func WritePermissionError(c *gin.Context, scope string) {
	c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
		Error:     "insufficient scope: " + scope,
		Code:      string(apperror.CodePermissionDenied),
		RequestID: c.GetString("request_id"),
	})
}

// WriteCustomError writes a custom error response with the specified status code,
// error message, and code.
func WriteCustomError(c *gin.Context, status int, message string, code string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Error:     message,
		Code:      code,
		RequestID: c.GetString("request_id"),
	})
}
