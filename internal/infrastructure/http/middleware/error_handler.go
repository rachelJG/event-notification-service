package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/errmap"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/httputil"
	"go.uber.org/zap"
)

// ErrorHandler is a middleware that centralizes error response handling.
// Handlers register errors via c.Error(err); this middleware runs after the
// handler and converts the last error into a structured JSON response with
// logging and metrics.
func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		httpErr := errmap.FromError(err)

		if logger != nil {
			logger.Error("request failed",
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

		httputil.WriteCustomError(c, httpErr.Status, errmap.Message(err), httpErr.Code)
	}
}
