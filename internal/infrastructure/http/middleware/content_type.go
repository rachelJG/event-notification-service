package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const jsonContentType = "application/json"

func RequireJSONContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch {
			contentType := c.GetHeader("Content-Type")
			if contentType == "" || !strings.HasPrefix(strings.ToLower(contentType), jsonContentType) {
				c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, gin.H{"error": "Content-Type must be application/json", "code": "invalid_argument", "request_id": c.GetString("request_id")})
				return
			}
		}
		c.Next()
	}
}
