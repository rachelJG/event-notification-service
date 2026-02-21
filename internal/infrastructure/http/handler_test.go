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
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"go.uber.org/zap"
)

type mockSubmitEvent struct {
	called       bool
	returnID     string
	returnErr    error
	receivedType string
	receivedKey  string
}

func (m *mockSubmitEvent) Handle(ctx context.Context, eventType string, payload []byte, idempotencyKey string) (string, error) {
	m.called = true
	m.receivedType = eventType
	m.receivedKey = idempotencyKey
	return m.returnID, m.returnErr
}

func TestSubmitEventHandlerMissingIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockSubmitEvent{returnID: "evt-1"}
	handler := Handler{SubmitEvent: mock, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
	if mock.called {
		t.Fatalf("expected use case not to be called")
	}
	assertErrorCode(t, resp, "invalid_argument")
}

func TestSubmitEventHandlerInvalidIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockSubmitEvent{returnID: "evt-1"}
	handler := Handler{SubmitEvent: mock, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

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
	if mock.called {
		t.Fatalf("expected use case not to be called")
	}
	assertErrorCode(t, resp, "invalid_argument")
}

func TestSubmitEventHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockSubmitEvent{returnID: "evt-1"}
	handler := Handler{SubmitEvent: mock, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

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
	if !mock.called {
		t.Fatalf("expected use case to be called")
	}
	if mock.receivedKey != "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a" {
		t.Fatalf("expected idempotency key to be passed to use case")
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if body["id"] != "evt-1" {
		t.Fatalf("expected id to be evt-1, got %s", body["id"])
	}
}

func TestSubmitEventHandlerUseCaseError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockSubmitEvent{returnErr: apperror.InvalidArgument("bad event", nil)}
	handler := Handler{SubmitEvent: mock, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestSubmitEventUnauthorizedWithoutAuthHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockSubmitEvent{returnID: "evt-1"}
	handler := Handler{SubmitEvent: mock, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
	if mock.called {
		t.Fatalf("expected use case not to be called")
	}
}

func TestSubmitEventRejectsNonJSONContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockSubmitEvent{returnID: "evt-1"}
	handler := Handler{SubmitEvent: mock, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

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
	if mock.called {
		t.Fatalf("expected use case not to be called")
	}
}

func TestSubmitEventRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockSubmitEvent{returnID: "evt-1"}
	handler := Handler{SubmitEvent: mock, Logger: zap.NewNop()}
	opts := testRouterOptions()
	opts.RateLimitRPS = 1
	opts.RateLimitBurst = 1
	router := NewRouter(handler, nil, zap.NewNop(), opts)

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

func assertErrorCode(t *testing.T, resp *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not parse response body: %v", err)
	}
	if body["code"] != wantCode {
		t.Fatalf("expected error code %q, got %q", wantCode, body["code"])
	}
}

func testRouterOptions() RouterOptions {
	return RouterOptions{
		JWTSecret:           "test-secret",
		MaxBodyBytes:        1 << 20,
		RateLimitRPS:        1000,
		RateLimitBurst:      1000,
		CORSAllowAllOrigins: true,
		CORSAllowedHeaders:  []string{"Origin", "Content-Length", "Content-Type", "Idempotency-Key", "Authorization"},
	}
}
