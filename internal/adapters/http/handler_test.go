package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rachelJG/event-notification-service/internal/config"
	"github.com/rachelJG/event-notification-service/internal/core/domain"
	"github.com/rachelJG/event-notification-service/internal/core/usecases"
	"go.uber.org/zap"
)

type fakeRepo struct {
	called bool
	event  domain.Event
}

func (r *fakeRepo) Create(ctx context.Context, event domain.Event) (string, error) {
	r.called = true
	r.event = event
	return "evt-1", nil
}

func TestSubmitEventHandlerMissingIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepo{}
	handler := Handler{SubmitEvent: usecases.SubmitEvent{Repo: repo}, Logger: zap.NewNop()}
	router := NewRouter(handler, zap.NewNop(), testConfig())

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
	if repo.called {
		t.Fatalf("expected repo not to be called")
	}
}

func TestSubmitEventHandlerInvalidIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepo{}
	handler := Handler{SubmitEvent: usecases.SubmitEvent{Repo: repo}, Logger: zap.NewNop()}
	router := NewRouter(handler, zap.NewNop(), testConfig())

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "not-a-uuid")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
	if repo.called {
		t.Fatalf("expected repo not to be called")
	}
}

func TestSubmitEventHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepo{}
	handler := Handler{SubmitEvent: usecases.SubmitEvent{Repo: repo}, Logger: zap.NewNop()}
	router := NewRouter(handler, zap.NewNop(), testConfig())

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.Code)
	}
	if !repo.called {
		t.Fatalf("expected repo to be called")
	}
	if repo.event.IdempotencyKey != "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a" {
		t.Fatalf("expected idempotency key to be passed to repo")
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if body["id"] != "evt-1" {
		t.Fatalf("expected id to be evt-1, got %s", body["id"])
	}
}

func TestSubmitEventRejectsNonJSONContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepo{}
	handler := Handler{SubmitEvent: usecases.SubmitEvent{Repo: repo}, Logger: zap.NewNop()}
	router := NewRouter(handler, zap.NewNop(), testConfig())

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", resp.Code)
	}
	if repo.called {
		t.Fatalf("expected repo not to be called")
	}
}

func TestSubmitEventRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRepo{}
	handler := Handler{SubmitEvent: usecases.SubmitEvent{Repo: repo}, Logger: zap.NewNop()}
	cfg := testConfig()
	cfg.RateLimitRPS = 1
	cfg.RateLimitBurst = 1
	router := NewRouter(handler, zap.NewNop(), cfg)

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "1.2.3.4:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected first request 202, got %d", resp.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req2.RemoteAddr = "1.2.3.4:1234"
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req2.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp2 := httptest.NewRecorder()

	router.ServeHTTP(resp2, req2)
	if resp2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", resp2.Code)
	}
}

func testJWT(t *testing.T, secret string) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "test-user",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func testConfig() config.Config {
	return config.Config{
		JWTSecret:         "test-secret",
		MaxBodyBytes:      1 << 20,
		RateLimitRPS:      1000,
		RateLimitBurst:    1000,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
