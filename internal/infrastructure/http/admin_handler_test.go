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
		IsActive: true,
	}
}

func adminTestRouter() (*gin.Engine, *fakeAPIKeyRepo) {
	gin.SetMode(gin.TestMode)
	repo := newFakeAPIKeyRepo(adminTestAPIKey())
	handler := Handler{EventService: &mockEventService{}, Logger: zap.NewNop()}
	adminHandler := AdminHandler{APIKeyRepo: repo, Logger: zap.NewNop()}
	opts := testRouterOptions()
	router := NewRouter(handler, adminHandler, nil, repo, zap.NewNop(), opts)
	return router, repo
}

func TestCreateAPIKeyHandlerSuccess(t *testing.T) {
	router, _ := adminTestRouter()

	body := `{"name":"my-service","scopes":["events:read","events:write"]}`
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

	body := `{"scopes":["events:read"]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestCreateAPIKeyHandlerKeyGenError(t *testing.T) {
	original := generateRawKeyFunc
	generateRawKeyFunc = func() (string, error) {
		return "", errors.New("entropy exhausted")
	}
	t.Cleanup(func() { generateRawKeyFunc = original })

	router, _ := adminTestRouter()

	body := `{"name":"my-service","scopes":["events:read"]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", resp.Code, resp.Body.String())
	}
}

func TestCreateAPIKeyHandlerKeyGenErrorNilLogger(t *testing.T) {
	original := generateRawKeyFunc
	generateRawKeyFunc = func() (string, error) {
		return "", errors.New("entropy exhausted")
	}
	t.Cleanup(func() { generateRawKeyFunc = original })

	gin.SetMode(gin.TestMode)
	repo := newFakeAPIKeyRepo(adminTestAPIKey())
	handler := Handler{EventService: &mockEventService{}, Logger: zap.NewNop()}
	adminHandler := AdminHandler{APIKeyRepo: repo, Logger: nil}
	router := NewRouter(handler, adminHandler, nil, repo, zap.NewNop(), testRouterOptions())

	body := `{"name":"my-service","scopes":["events:read"]}`
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
	handler := Handler{EventService: &mockEventService{}, Logger: zap.NewNop()}
	adminHandler := AdminHandler{APIKeyRepo: failRepo, Logger: zap.NewNop()}
	router := NewRouter(handler, adminHandler, nil, failRepo, zap.NewNop(), testRouterOptions())

	body := `{"name":"fail-key","scopes":["events:read"]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api-keys", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestCreateAPIKeyHandlerNilLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	failRepo := &failingAPIKeyRepo{fakeAPIKeyRepo: newFakeAPIKeyRepo(adminTestAPIKey()), createErr: apperror.Internal("db down", nil)}
	handler := Handler{EventService: &mockEventService{}, Logger: zap.NewNop()}
	adminHandler := AdminHandler{APIKeyRepo: failRepo, Logger: nil}
	router := NewRouter(handler, adminHandler, nil, failRepo, zap.NewNop(), testRouterOptions())

	body := `{"name":"fail-key","scopes":["events:read"]}`
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

	var result map[string][]apiKeyListItem
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(result["keys"]) == 0 {
		t.Fatal("expected at least one key in list")
	}
}

func TestListAPIKeysHandlerRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	failRepo := &failingAPIKeyRepo{fakeAPIKeyRepo: newFakeAPIKeyRepo(adminTestAPIKey()), listErr: apperror.Internal("db down", nil)}
	handler := Handler{EventService: &mockEventService{}, Logger: zap.NewNop()}
	adminHandler := AdminHandler{APIKeyRepo: failRepo, Logger: zap.NewNop()}
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

func TestListAPIKeysHandlerNilLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	failRepo := &failingAPIKeyRepo{fakeAPIKeyRepo: newFakeAPIKeyRepo(adminTestAPIKey()), listErr: apperror.Internal("db down", nil)}
	handler := Handler{EventService: &mockEventService{}, Logger: zap.NewNop()}
	adminHandler := AdminHandler{APIKeyRepo: failRepo, Logger: nil}
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
		ID:       "revoke-me",
		KeyHash:  middleware.HashAPIKey("some-other-key"),
		Name:     "to-revoke",
		Scopes:   []string{"events:read"},
		IsActive: true,
	}
	_ = repo.Create(context.Background(), key)

	req := httptest.NewRequest(http.MethodDelete, "/admin/api-keys/revoke-me", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", resp.Code, resp.Body.String())
	}

	var result map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if result["id"] != "revoke-me" {
		t.Errorf("expected id revoke-me, got %q", result["id"])
	}
}

func TestRevokeAPIKeyHandlerNotFound(t *testing.T) {
	router, _ := adminTestRouter()

	req := httptest.NewRequest(http.MethodDelete, "/admin/api-keys/nonexistent", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", resp.Code, resp.Body.String())
	}
}

func TestRevokeAPIKeyHandlerNilLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeAPIKeyRepo(adminTestAPIKey())
	handler := Handler{EventService: &mockEventService{}, Logger: zap.NewNop()}
	adminHandler := AdminHandler{APIKeyRepo: repo, Logger: nil}
	router := NewRouter(handler, adminHandler, nil, repo, zap.NewNop(), testRouterOptions())

	req := httptest.NewRequest(http.MethodDelete, "/admin/api-keys/nonexistent", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
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
		{http.MethodDelete, "/admin/api-keys/some-id"},
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
