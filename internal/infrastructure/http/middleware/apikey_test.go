package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeAPIKeyRepo is a stub implementation of ports.APIKeyRepository for tests.
type fakeAPIKeyRepo struct {
	keys          map[string]entities.APIKey // keyed by key_hash
	createErr     error
	getByHashErr  error
	listErr       error
	revokeErr     error
	updateLastErr error

	mu             sync.Mutex
	lastUsedCalls  []string // IDs passed to UpdateLastUsed
}

func (f *fakeAPIKeyRepo) Create(_ context.Context, key entities.APIKey) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.keys == nil {
		f.keys = make(map[string]entities.APIKey)
	}
	f.keys[key.KeyHash] = key
	return nil
}

func (f *fakeAPIKeyRepo) GetByHash(_ context.Context, keyHash string) (entities.APIKey, error) {
	if f.getByHashErr != nil {
		return entities.APIKey{}, f.getByHashErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	k, ok := f.keys[keyHash]
	if !ok {
		return entities.APIKey{}, apperror.NotFound("api key not found", nil)
	}
	return k, nil
}

func (f *fakeAPIKeyRepo) List(_ context.Context) ([]entities.APIKey, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []entities.APIKey
	for _, k := range f.keys {
		out = append(out, k)
	}
	return out, nil
}

func (f *fakeAPIKeyRepo) Revoke(_ context.Context, id string) error {
	if f.revokeErr != nil {
		return f.revokeErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for hash, k := range f.keys {
		if k.ID == id {
			k.IsActive = false
			f.keys[hash] = k
			return nil
		}
	}
	return apperror.NotFound("api key not found", nil)
}

func (f *fakeAPIKeyRepo) UpdateLastUsed(_ context.Context, id string) error {
	if f.updateLastErr != nil {
		return f.updateLastErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastUsedCalls = append(f.lastUsedCalls, id)
	return nil
}

func (f *fakeAPIKeyRepo) getLastUsedCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]string, len(f.lastUsedCalls))
	copy(cp, f.lastUsedCalls)
	return cp
}

// seedKey adds a key to the fake repo and returns the raw key string.
func seedKey(t *testing.T, repo *fakeAPIKeyRepo, id, name string, scopes []string, active bool) string {
	t.Helper()
	rawKey := "test-key-" + id
	hash := HashAPIKey(rawKey)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.keys == nil {
		repo.keys = make(map[string]entities.APIKey)
	}
	repo.keys[hash] = entities.APIKey{
		ID:       id,
		KeyHash:  hash,
		Name:     name,
		Scopes:   scopes,
		IsActive: active,
	}
	return rawKey
}

func newAPIKeyRouter(opts APIKeyOptions) *gin.Engine {
	r := gin.New()
	r.POST("/test", APIKeyAuth(opts), func(c *gin.Context) {
		name, _ := c.Get(ContextKeyAPIKeyName)
		c.JSON(http.StatusOK, gin.H{"api_key_name": name})
	})
	return r
}

func parseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), "failed to parse response body")
	return body
}

func TestAPIKeyAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions)
		setHeader  func(req *http.Request, rawKey string)
		wantStatus int
		wantCode   string
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "valid key with required scopes",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				repo := &fakeAPIKeyRepo{}
				rawKey := seedKey(t, repo, "k1", "my-service", []string{"events:write", "events:read"}, true)
				opts := APIKeyOptions{
					Repo:           repo,
					RequiredScopes: []string{"events:write"},
				}
				return repo, rawKey, opts
			},
			setHeader:  func(req *http.Request, rawKey string) { req.Header.Set("X-API-Key", rawKey) },
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "my-service", body["api_key_name"], "expected api_key_name in response")
			},
		},
		{
			name: "missing X-API-Key header",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				repo := &fakeAPIKeyRepo{}
				opts := APIKeyOptions{Repo: repo}
				return repo, "", opts
			},
			setHeader:  func(req *http.Request, _ string) { /* no header */ },
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthenticated",
		},
		{
			name: "empty X-API-Key header",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				repo := &fakeAPIKeyRepo{}
				opts := APIKeyOptions{Repo: repo}
				return repo, "", opts
			},
			setHeader:  func(req *http.Request, _ string) { req.Header.Set("X-API-Key", "") },
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthenticated",
		},
		{
			name: "whitespace-only X-API-Key header",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				repo := &fakeAPIKeyRepo{}
				opts := APIKeyOptions{Repo: repo}
				return repo, "", opts
			},
			setHeader:  func(req *http.Request, _ string) { req.Header.Set("X-API-Key", "   ") },
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthenticated",
		},
		{
			name: "unknown key returns 401",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				repo := &fakeAPIKeyRepo{}
				opts := APIKeyOptions{Repo: repo}
				return repo, "nonexistent-key", opts
			},
			setHeader:  func(req *http.Request, rawKey string) { req.Header.Set("X-API-Key", rawKey) },
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthenticated",
		},
		{
			name: "revoked key returns 401",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				repo := &fakeAPIKeyRepo{}
				rawKey := seedKey(t, repo, "k2", "revoked-service", []string{"events:write"}, false)
				opts := APIKeyOptions{Repo: repo}
				return repo, rawKey, opts
			},
			setHeader:  func(req *http.Request, rawKey string) { req.Header.Set("X-API-Key", rawKey) },
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthenticated",
		},
		{
			name: "missing required scope returns 403",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				repo := &fakeAPIKeyRepo{}
				rawKey := seedKey(t, repo, "k3", "limited-service", []string{"events:read"}, true)
				opts := APIKeyOptions{
					Repo:           repo,
					RequiredScopes: []string{"events:write"},
				}
				return repo, rawKey, opts
			},
			setHeader:  func(req *http.Request, rawKey string) { req.Header.Set("X-API-Key", rawKey) },
			wantStatus: http.StatusForbidden,
			wantCode:   "permission_denied",
		},
		{
			name: "nil repo returns 500",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				opts := APIKeyOptions{Repo: nil}
				return nil, "any-key", opts
			},
			setHeader:  func(req *http.Request, rawKey string) { req.Header.Set("X-API-Key", rawKey) },
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal",
		},
		{
			name: "no required scopes passes with any active key",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				repo := &fakeAPIKeyRepo{}
				rawKey := seedKey(t, repo, "k4", "any-scope-svc", []string{}, true)
				opts := APIKeyOptions{
					Repo:           repo,
					RequiredScopes: nil,
				}
				return repo, rawKey, opts
			},
			setHeader:  func(req *http.Request, rawKey string) { req.Header.Set("X-API-Key", rawKey) },
			wantStatus: http.StatusOK,
		},
		{
			name: "multiple required scopes all present",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				repo := &fakeAPIKeyRepo{}
				rawKey := seedKey(t, repo, "k5", "full-access", []string{"events:read", "events:write", "admin"}, true)
				opts := APIKeyOptions{
					Repo:           repo,
					RequiredScopes: []string{"events:read", "events:write", "admin"},
				}
				return repo, rawKey, opts
			},
			setHeader:  func(req *http.Request, rawKey string) { req.Header.Set("X-API-Key", rawKey) },
			wantStatus: http.StatusOK,
		},
		{
			name: "multiple required scopes one missing",
			setup: func(t *testing.T) (*fakeAPIKeyRepo, string, APIKeyOptions) {
				repo := &fakeAPIKeyRepo{}
				rawKey := seedKey(t, repo, "k6", "partial", []string{"events:read", "events:write"}, true)
				opts := APIKeyOptions{
					Repo:           repo,
					RequiredScopes: []string{"events:read", "admin"},
				}
				return repo, rawKey, opts
			},
			setHeader:  func(req *http.Request, rawKey string) { req.Header.Set("X-API-Key", rawKey) },
			wantStatus: http.StatusForbidden,
			wantCode:   "permission_denied",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, rawKey, opts := tc.setup(t)
			router := newAPIKeyRouter(opts)

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			tc.setHeader(req, rawKey)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code, "unexpected status code")

			if tc.wantCode != "" {
				body := parseBody(t, w)
				assert.Equal(t, tc.wantCode, body["code"], "unexpected error code")
			}
			if tc.checkBody != nil {
				body := parseBody(t, w)
				tc.checkBody(t, body)
			}
		})
	}
}

func TestAPIKeyAuth_StoresNameInContext(t *testing.T) {
	t.Parallel()

	repo := &fakeAPIKeyRepo{}
	rawKey := seedKey(t, repo, "ctx-1", "context-test-svc", []string{"events:write"}, true)

	var capturedName string
	r := gin.New()
	r.POST("/test", APIKeyAuth(APIKeyOptions{
		Repo:           repo,
		RequiredScopes: []string{"events:write"},
	}), func(c *gin.Context) {
		capturedName = c.GetString(ContextKeyAPIKeyName)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("X-API-Key", rawKey)
	r.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, "context-test-svc", capturedName)
}

func TestAPIKeyAuth_UpdatesLastUsedAsync(t *testing.T) {
	t.Parallel()

	repo := &fakeAPIKeyRepo{}
	rawKey := seedKey(t, repo, "async-1", "async-svc", []string{}, true)

	router := newAPIKeyRouter(APIKeyOptions{Repo: repo})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("X-API-Key", rawKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Wait briefly for the async goroutine to complete.
	assert.Eventually(t, func() bool {
		calls := repo.getLastUsedCalls()
		return len(calls) == 1 && calls[0] == "async-1"
	}, 2*time.Second, 10*time.Millisecond, "expected UpdateLastUsed to be called with key ID")
}

func TestAPIKeyAuth_WithLogger(t *testing.T) {
	t.Parallel()

	logger, _ := zap.NewDevelopment()

	t.Run("logs success", func(t *testing.T) {
		t.Parallel()
		repo := &fakeAPIKeyRepo{}
		rawKey := seedKey(t, repo, "log-1", "log-svc", []string{"events:write"}, true)

		router := newAPIKeyRouter(APIKeyOptions{
			Repo:           repo,
			RequiredScopes: []string{"events:write"},
			Logger:         logger,
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("X-API-Key", rawKey)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("logs key not found", func(t *testing.T) {
		t.Parallel()
		repo := &fakeAPIKeyRepo{}
		router := newAPIKeyRouter(APIKeyOptions{
			Repo:   repo,
			Logger: logger,
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("X-API-Key", "unknown-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("logs revoked key", func(t *testing.T) {
		t.Parallel()
		repo := &fakeAPIKeyRepo{}
		rawKey := seedKey(t, repo, "log-2", "revoked-svc", []string{}, false)

		router := newAPIKeyRouter(APIKeyOptions{
			Repo:   repo,
			Logger: logger,
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("X-API-Key", rawKey)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("logs missing scope", func(t *testing.T) {
		t.Parallel()
		repo := &fakeAPIKeyRepo{}
		rawKey := seedKey(t, repo, "log-3", "limited-svc", []string{"events:read"}, true)

		router := newAPIKeyRouter(APIKeyOptions{
			Repo:           repo,
			RequiredScopes: []string{"admin"},
			Logger:         logger,
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("X-API-Key", rawKey)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestHashAPIKey(t *testing.T) {
	t.Parallel()

	t.Run("deterministic", func(t *testing.T) {
		t.Parallel()
		h1 := HashAPIKey("my-secret-key")
		h2 := HashAPIKey("my-secret-key")
		assert.Equal(t, h1, h2, "same input must produce same hash")
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		t.Parallel()
		h1 := HashAPIKey("key-a")
		h2 := HashAPIKey("key-b")
		assert.NotEqual(t, h1, h2, "different inputs must produce different hashes")
	})

	t.Run("returns 64-char hex string", func(t *testing.T) {
		t.Parallel()
		h := HashAPIKey("anything")
		assert.Len(t, h, 64, "SHA-256 hex digest should be 64 characters")
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		h := HashAPIKey("")
		assert.Len(t, h, 64, "even empty input should produce a valid hash")
	})
}

func TestWritePermissionError(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set("request_id", "req-123")
		writePermissionError(c, "admin")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	body := parseBody(t, w)
	assert.Equal(t, "insufficient scope: admin", body["error"])
	assert.Equal(t, "permission_denied", body["code"])
	assert.Equal(t, "req-123", body["request_id"])
}
