package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/httputil"
	"go.uber.org/zap"
)

// ContextKeySubject is the gin context key where the JWT subject claim is stored.
const ContextKeySubject = "jwt_subject"

// JWTOptions configures JWT authentication behaviour.
type JWTOptions struct {
	Secret   string
	Issuer   string // optional; validated when non-empty
	Audience string // optional; validated when non-empty
	Logger   *zap.Logger
}

// JWTAuth validates a Bearer JWT using HS256. It enforces expiry, not-before,
// and optionally issuer / audience claims. On success it stores the token
// subject in the gin context under ContextKeySubject.
func JWTAuth(opts JWTOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if opts.Secret == "" {
			httputil.WriteError(c, apperror.Internal("jwt secret not configured", nil))
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			httputil.WriteError(c, apperror.Unauthenticated("missing or invalid authorization header", nil))
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
			if opts.Logger != nil {
				opts.Logger.Warn("auth failed",
					zap.String("event", "auth"),
					zap.String("reason", err.Error()),
					zap.String("remote_ip", c.ClientIP()),
					zap.String("request_id", c.GetString("request_id")),
				)
			}
			httputil.WriteError(c, apperror.Unauthenticated("invalid token", err))
			return
		}

		if opts.Logger != nil {
			opts.Logger.Info("auth success",
				zap.String("event", "auth"),
				zap.String("sub", claims.Subject),
				zap.String("remote_ip", c.ClientIP()),
				zap.String("request_id", c.GetString("request_id")),
			)
		}
		c.Set(ContextKeySubject, claims.Subject)
		c.Next()
	}
}
