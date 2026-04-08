package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/middleware"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"go.uber.org/zap"
)

// adminTestAPIKey returns an active API key with "admin" scope for admin endpoint tests.
func adminTestAPIKey() entities.APIKey {
	return entities.APIKey{
		ID:       "admin-key-1",
		KeyHash:  middleware.HashAPIKey(testRawAPIKey),
		Name:     "admin-test-key",
		Scopes:   []string{"admin"},
		Metadata: map[string]string{"client_id": "test-admin"},
		IsActive: true,
	}
}

func adminTestRouter() (*gin.Engine, *fakeAPIKeyRepo) {
	gin.SetMode(gin.TestMode)
	repo := newFakeAPIKeyRepo(adminTestAPIKey())
	handler := Handler{EventService: &mockEventService{}}
	adminHandler := AdminHandler{APIKeyRepo: repo}
	opts := testRouterOptions()
	router := NewRouter(handler, adminHandler, nil, repo, zap.NewNop(), opts)
	return router, repo
}

func TestCreateAPIKeyHandlerSuccess(t *testing.T) {
	router, _ := adminTestRouter()

	body := `{"name":"my-service","scopes":["events:read","events:write"],"metadata":{"client_id":"acme-corp","organization":"Acme Corp"}}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", resp.Code, resp.Body.String())
	}

	var result createAPIKeyResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if result.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if result.Name != "my-service" {
		t.Errorf("expected name my-service, got %q", result.Name)
	}
	if result.Key == "" {
		t.Fatal("expected raw key to be returned on creation")
	}
	if len(result.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(result.Scopes))
	}
	if result.Metadata["client_id"] != "acme-corp" {
		t.Errorf("expected client_id acme-corp, got %q", result.Metadata["client_id"])
	}
}

func TestCreateAPIKeyHandlerInvalidJSON(t *testing.T) {
	router, _ := adminTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestCreateAPIKeyHandlerMissingName(t *testing.T) {
	router, _ := adminTestRouter()

	body := `{"scopes":["events:read"],"metadata":{"client_id":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestCreateAPIKeyHandlerMissingClientID(t *testing.T) {
	router, _ := adminTestRouter()

	body := `{"name":"my-service","scopes":["events:read"]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", resp.Code, resp.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("client_id")) {
		t.Error("expected error message to mention client_id")
	}
}

func TestCreateAPIKeyHandlerEmptyClientID(t *testing.T) {
	router, _ := adminTestRouter()

	body := `{"name":"my-service","scopes":["events:read"],"metadata":{"client_id":""}}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", resp.Code, resp.Body.String())
	}
}

func TestCreateAPIKeyHandlerKeyGenError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeAPIKeyRepo(adminTestAPIKey())
	handler := Handler{EventService: &mockEventService{}}
	adminHandler := AdminHandler{
		APIKeyRepo: repo,
		KeyGenerator: func() (string, error) {
			return "", errors.New("entropy exhausted")
		},
	}
	router := NewRouter(handler, adminHandler, nil, repo, zap.NewNop(), testRouterOptions())

	body := `{"name":"my-service","scopes":["events:read"],"metadata":{"client_id":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestCreateAPIKeyHandlerRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	failRepo := &failingAPIKeyRepo{fakeAPIKeyRepo: newFakeAPIKeyRepo(adminTestAPIKey()), createErr: apperror.Internal("db down", nil)}
	handler := Handler{EventService: &mockEventService{}}
	adminHandler := AdminHandler{APIKeyRepo: failRepo}
	router := NewRouter(handler, adminHandler, nil, failRepo, zap.NewNop(), testRouterOptions())

	body := `{"name":"fail-key","scopes":["events:read"],"metadata":{"client_id":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestListAPIKeysHandlerSuccess(t *testing.T) {
	router, _ := adminTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/admin/api-keys", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", resp.Code, resp.Body.String())
	}

	var result listAPIKeysResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(result.Keys) == 0 {
		t.Fatal("expected at least one key in list")
	}
}

func TestListAPIKeysHandlerRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	failRepo := &failingAPIKeyRepo{fakeAPIKeyRepo: newFakeAPIKeyRepo(adminTestAPIKey()), listErr: apperror.Internal("db down", nil)}
	handler := Handler{EventService: &mockEventService{}}
	adminHandler := AdminHandler{APIKeyRepo: failRepo}
	router := NewRouter(handler, adminHandler, nil, failRepo, zap.NewNop(), testRouterOptions())

	req := httptest.NewRequest(http.MethodGet, "/admin/api-keys", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestRevokeAPIKeyHandlerSuccess(t *testing.T) {
	router, repo := adminTestRouter()

	// Create a key to revoke
	key := entities.APIKey{
		ID:       "550e8400-e29b-41d4-a716-446655440000",
		KeyHash:  middleware.HashAPIKey("some-other-key"),
		Name:     "to-revoke",
		Scopes:   []string{"events:read"},
		IsActive: true,
	}
	_ = repo.Create(context.Background(), key)

	req := httptest.NewRequest(http.MethodDelete, "/admin/api-keys/550e8400-e29b-41d4-a716-446655440000", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", resp.Code, resp.Body.String())
	}

	var result revokeAPIKeyResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if result.ID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected id 550e8400-e29b-41d4-a716-446655440000, got %q", result.ID)
	}
}

func TestRevokeAPIKeyHandlerInvalidUUID(t *testing.T) {
	router, _ := adminTestRouter()

	req := httptest.NewRequest(http.MethodDelete, "/admin/api-keys/not-a-uuid", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", resp.Code, resp.Body.String())
	}
}

func TestRevokeAPIKeyHandlerNotFound(t *testing.T) {
	router, _ := adminTestRouter()

	req := httptest.NewRequest(http.MethodDelete, "/admin/api-keys/123e4567-e89b-12d3-a456-426614174000", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", resp.Code, resp.Body.String())
	}
}

func TestAdminEndpointsRequireAuth(t *testing.T) {
	router, _ := adminTestRouter()

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/admin/api-keys"},
		{http.MethodGet, "/admin/api-keys"},
		{http.MethodDelete, "/admin/api-keys/123e4567-e89b-12d3-a456-426614174000"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 without auth, got %d", resp.Code)
			}
		})
	}
}

func TestGenerateRawKey(t *testing.T) {
	key, err := generateRawKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) != 64 {
		t.Errorf("expected 64-char hex string, got len %d", len(key))
	}

	// Verify uniqueness (two calls should produce different keys)
	key2, err := generateRawKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == key2 {
		t.Error("expected unique keys from two calls")
	}
}

// failingAPIKeyRepo wraps fakeAPIKeyRepo but returns errors for specific operations.
type failingAPIKeyRepo struct {
	*fakeAPIKeyRepo
	createErr error
	listErr   error
}

func (r *failingAPIKeyRepo) Create(_ context.Context, _ entities.APIKey) error {
	if r.createErr != nil {
		return r.createErr
	}
	return nil
}

func (r *failingAPIKeyRepo) List(_ context.Context) ([]entities.APIKey, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.fakeAPIKeyRepo.List(context.Background())
}
