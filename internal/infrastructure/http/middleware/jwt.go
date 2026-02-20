package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	apperror "github.com/rachelJG/event-notification-service/internal/domain/errors"
)

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			writeAuthError(c, apperror.Internal("jwt secret not configured", nil))
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			writeAuthError(c, apperror.Unauthenticated("missing or invalid authorization header", nil))
			return
		}

		tokenString := strings.TrimPrefix(auth, "Bearer ")
		_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, apperror.Unauthenticated("invalid token", jwt.ErrTokenSignatureInvalid)
			}
			return []byte(secret), nil
		})
		if err != nil {
			writeAuthError(c, apperror.Unauthenticated("invalid token", err))
			return
		}

		c.Next()
	}
}

func writeAuthError(c *gin.Context, err error) {
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
		}
	}
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}
