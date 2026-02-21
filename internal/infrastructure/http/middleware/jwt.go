package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

// ContextKeySubject is the gin context key where the JWT subject claim is stored.
const ContextKeySubject = "jwt_subject"

// JWTOptions configures JWT authentication behaviour.
type JWTOptions struct {
	Secret   string
	Issuer   string // optional; validated when non-empty
	Audience string // optional; validated when non-empty
}

// JWTAuth validates a Bearer JWT using HS256. It enforces expiry, not-before,
// and optionally issuer / audience claims. On success it stores the token
// subject in the gin context under ContextKeySubject.
func JWTAuth(opts JWTOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if opts.Secret == "" {
			writeAuthError(c, apperror.Internal("jwt secret not configured", nil))
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			writeAuthError(c, apperror.Unauthenticated("missing or invalid authorization header", nil))
			return
		}

		tokenString := strings.TrimPrefix(auth, "Bearer ")

		parserOpts := []jwt.ParserOption{
			jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithExpirationRequired(),
		}
		if opts.Issuer != "" {
			parserOpts = append(parserOpts, jwt.WithIssuer(opts.Issuer))
		}
		if opts.Audience != "" {
			parserOpts = append(parserOpts, jwt.WithAudience(opts.Audience))
		}

		var claims jwt.RegisteredClaims
		_, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(opts.Secret), nil
		}, parserOpts...)
		if err != nil {
			writeAuthError(c, apperror.Unauthenticated("invalid token", err))
			return
		}

		c.Set(ContextKeySubject, claims.Subject)
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
