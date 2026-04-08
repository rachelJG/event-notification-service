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

// createAPIKeyRequest represents the request body for creating an API key
type createAPIKeyRequest struct {
	Name     string            `json:"name" binding:"required" example:"Acme Corp - Production"`      // Human-readable name for the API key
	Scopes   []string          `json:"scopes" binding:"required" example:"events:write,events:read"`  // List of scopes (events:write, events:read, admin)
	Metadata map[string]string `json:"metadata" example:"client_id:acme-corp,organization:Acme Corp"` // Key metadata (client_id is required)
}

// createAPIKeyResponse represents the response after creating an API key
type createAPIKeyResponse struct {
	ID       string            `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"` // API key ID (UUID)
	Name     string            `json:"name" example:"Acme Corp - Production"`             // Human-readable name
	Scopes   []string          `json:"scopes" example:"events:write,events:read"`         // List of scopes
	Metadata map[string]string `json:"metadata,omitempty" example:"client_id:acme-corp"`  // Key metadata
	Key      string            `json:"key" example:"abc123...xyz"`                        // Raw API key (returned only on creation)
}

// apiKeyListItem represents an API key in the list response
type apiKeyListItem struct {
	ID         string            `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`     // API key ID (UUID)
	Name       string            `json:"name" example:"Acme Corp - Production"`                 // Human-readable name
	Scopes     []string          `json:"scopes" example:"events:write,events:read"`             // List of scopes
	Metadata   map[string]string `json:"metadata,omitempty" example:"client_id:acme-corp"`      // Key metadata
	IsActive   bool              `json:"is_active" example:"true"`                              // Whether the key is active
	CreatedAt  time.Time         `json:"created_at" example:"2024-01-15T10:30:00Z"`             // When the key was created
	LastUsedAt *time.Time        `json:"last_used_at,omitempty" example:"2024-01-15T12:00:00Z"` // When the key was last used
}

type listAPIKeysResponse struct {
	Keys []apiKeyListItem `json:"keys"`
}

type revokeAPIKeyResponse struct {
	Message string `json:"message" example:"api key revoked"`
	ID      string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// CreateAPIKeyHandler godoc
//
//	@Summary		Create a new API key
//	@Description	Generates a new API key with specified scopes and metadata. Returns the raw key only once.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			X-API-Key	header		string					true	"API Key with admin scope"
//	@Param			request		body		createAPIKeyRequest		true	"API key details"
//	@Success		201			{object}	createAPIKeyResponse
//	@Failure		400			{object}	map[string]interface{}	"Invalid request (bad JSON, missing client_id in metadata)"
//	@Failure		401			{object}	map[string]interface{}	"Missing or invalid API key"
//	@Router			/admin/api-keys [post]
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

// ListAPIKeysHandler godoc
//
//	@Summary		List all API keys
//	@Description	Returns all API keys with their metadata (excluding raw keys)
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			X-API-Key	header		string	true	"API Key with admin scope"
//	@Success		200			{object}	listAPIKeysResponse
//	@Failure		401			{object}	map[string]interface{}	"Missing or invalid API key"
//	@Router			/admin/api-keys [get]
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
	c.JSON(http.StatusOK, listAPIKeysResponse{Keys: items})
}

// RevokeAPIKeyHandler godoc
//
//	@Summary		Revoke an API key
//	@Description	Deactivates an API key by setting is_active to false
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Param			X-API-Key	header		string	true	"API Key with admin scope"
//	@Param			id			path		string	true	"API Key ID (UUID)"
//	@Success		200			{object}	revokeAPIKeyResponse
//	@Failure		400			{object}	map[string]interface{}	"Invalid API key ID format"
//	@Failure		401			{object}	map[string]interface{}	"Missing or invalid API key"
//	@Failure		404			{object}	map[string]interface{}	"API key not found"
//	@Router			/admin/api-keys/{id} [delete]
func (h AdminHandler) RevokeAPIKeyHandler(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		_ = c.Error(apperror.InvalidArgument("invalid API key ID format", err))
		return
	}
	if err := h.APIKeyRepo.Revoke(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, revokeAPIKeyResponse{Message: "api key revoked", ID: id})
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
