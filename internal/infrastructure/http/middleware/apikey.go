package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	domainports "github.com/rachelJG/event-notification-service/internal/domain/ports"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/httputil"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"go.uber.org/zap"
)

// ContextKeyAPIKeyName is the gin context key where the API key name is stored.
const ContextKeyAPIKeyName = "api_key_name"

// ContextKeyClientID is the gin context key where the client_id from metadata is stored.
const ContextKeyClientID = "client_id"

// APIKeyOptions configures API Key authentication behaviour.
type APIKeyOptions struct {
	Repo           domainports.APIKeyRepository
	RequiredScopes []string // scopes that the key must have
	Logger         *zap.Logger
}

// APIKeyAuth validates an API key sent via the X-API-Key header. It looks up
// the SHA-256 hash of the provided key in the database, checks that the key is
// active and has the required scopes, and stores the key name in the gin context
// for audit logging.
func APIKeyAuth(opts APIKeyOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		if opts.Repo == nil {
			httputil.WriteError(c, apperror.Internal("api key repository not configured", nil))
			return
		}

		rawKey := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if rawKey == "" {
			httputil.WriteError(c, apperror.Unauthenticated("missing or invalid X-API-Key header", nil))
			return
		}

		keyHash := HashAPIKey(rawKey)

		apiKey, err := opts.Repo.GetByHash(c.Request.Context(), keyHash)
		if err != nil {
			logAuthFailure(opts.Logger, "key not found", c)
			httputil.WriteError(c, apperror.Unauthenticated("invalid api key", nil))
			return
		}

		if !apiKey.IsActive {
			logAuthFailure(opts.Logger, "key revoked", c)
			httputil.WriteError(c, apperror.Unauthenticated("api key has been revoked", nil))
			return
		}

		for _, scope := range opts.RequiredScopes {
			if !apiKey.HasScope(scope) {
				logAuthFailure(opts.Logger, "missing scope: "+scope, c)
				httputil.WritePermissionError(c, scope)
				return
			}
		}

		if opts.Logger != nil {
			opts.Logger.Info("auth success",
				zap.String("event", "auth"),
				zap.String("api_key_name", apiKey.Name),
				zap.String("client_id", apiKey.ClientID()),
				zap.String("remote_ip", c.ClientIP()),
				zap.String("request_id", c.GetString("request_id")),
			)
		}

		c.Set(ContextKeyAPIKeyName, apiKey.Name)
		c.Set(ContextKeyClientID, apiKey.ClientID())

		// Update last_used_at asynchronously to avoid adding latency.
		go func(repo domainports.APIKeyRepository, id string) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = repo.UpdateLastUsed(ctx, id)
		}(opts.Repo, apiKey.ID)

		c.Next()
	}
}

// HashAPIKey computes the SHA-256 hex digest of a raw API key string.
// Exported so that the genkey CLI and tests can use the same hashing logic.
func HashAPIKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func logAuthFailure(logger *zap.Logger, reason string, c *gin.Context) {
	if logger == nil {
		return
	}
	logger.Warn("auth failed",
		zap.String("event", "auth"),
		zap.String("reason", reason),
		zap.String("remote_ip", c.ClientIP()),
		zap.String("request_id", c.GetString("request_id")),
	)
}

