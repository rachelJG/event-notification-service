package httpadapter

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	domainports "github.com/rachelJG/event-notification-service/internal/domain/ports"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/middleware"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

// AdminHandler provides HTTP endpoints for API key management.
type AdminHandler struct {
	APIKeyRepo   domainports.APIKeyRepository
	KeyGenerator func() (string, error) // nil defaults to generateRawKey
}

type createAPIKeyRequest struct {
	Name     string            `json:"name" binding:"required"`
	Scopes   []string          `json:"scopes" binding:"required"`
	Metadata map[string]string `json:"metadata"`
}

type createAPIKeyResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Scopes   []string          `json:"scopes"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Key      string            `json:"key"` // returned only on creation
}

type apiKeyListItem struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Scopes     []string          `json:"scopes"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	IsActive   bool              `json:"is_active"`
	CreatedAt  time.Time         `json:"created_at"`
	LastUsedAt *time.Time        `json:"last_used_at,omitempty"`
}

// CreateAPIKeyHandler generates a new API key, stores its SHA-256 hash, and
// returns the raw key to the caller exactly once.
func (h AdminHandler) CreateAPIKeyHandler(c *gin.Context) {
	var req createAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.InvalidArgument("invalid JSON body", err))
		return
	}

	if req.Metadata == nil || req.Metadata["client_id"] == "" {
		_ = c.Error(apperror.InvalidArgument("metadata.client_id is required", nil))
		return
	}

	genKey := h.KeyGenerator
	if genKey == nil {
		genKey = generateRawKey
	}
	rawKey, err := genKey()
	if err != nil {
		_ = c.Error(apperror.Internal("failed to generate key", err))
		return
	}

	id := uuid.NewString()
	keyHash := middleware.HashAPIKey(rawKey)

	apiKey := entities.APIKey{
		ID:        id,
		KeyHash:   keyHash,
		Name:      req.Name,
		Scopes:    req.Scopes,
		Metadata:  req.Metadata,
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.APIKeyRepo.Create(c.Request.Context(), apiKey); err != nil {
		_ = c.Error(apperror.Internal("failed to create key", err))
		return
	}

	c.JSON(http.StatusCreated, createAPIKeyResponse{
		ID:       id,
		Name:     req.Name,
		Scopes:   req.Scopes,
		Metadata: req.Metadata,
		Key:      rawKey,
	})
}

// ListAPIKeysHandler returns all API keys (without hash values).
func (h AdminHandler) ListAPIKeysHandler(c *gin.Context) {
	keys, err := h.APIKeyRepo.List(c.Request.Context())
	if err != nil {
		_ = c.Error(apperror.Internal("failed to list keys", err))
		return
	}

	items := make([]apiKeyListItem, 0, len(keys))
	for _, k := range keys {
		items = append(items, apiKeyListItem{
			ID:         k.ID,
			Name:       k.Name,
			Scopes:     k.Scopes,
			Metadata:   k.Metadata,
			IsActive:   k.IsActive,
			CreatedAt:  k.CreatedAt,
			LastUsedAt: k.LastUsedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"keys": items})
}

// RevokeAPIKeyHandler deactivates an API key by its ID.
func (h AdminHandler) RevokeAPIKeyHandler(c *gin.Context) {
	id := c.Param("id")
	if err := h.APIKeyRepo.Revoke(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "api key revoked", "id": id})
}

// generateRawKey creates a 32-byte (256-bit) cryptographically random key
// and returns it as a 64-character hex string.
func generateRawKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
