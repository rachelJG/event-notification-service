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
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
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

type mockGetEvent struct {
	returnEvent entities.Event
	returnErr   error
}

func (m *mockGetEvent) Handle(ctx context.Context, id string) (entities.Event, error) {
	return m.returnEvent, m.returnErr
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

func TestSubmitEventHandlerLocationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockSubmitEvent{returnID: "evt-abc"}
	handler := Handler{SubmitEvent: mock, GetEvent: &mockGetEvent{}, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "7c9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.Code)
	}
	loc := resp.Header().Get("Location")
	if loc != "/api/v1/events/evt-abc" {
		t.Fatalf("expected Location header /api/v1/events/evt-abc, got %q", loc)
	}
}

func TestGetEventHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	evt := entities.Event{ID: "evt-1", Type: "UserRegistered", Payload: []byte(`{"user_id":"1"}`)}
	handler := Handler{SubmitEvent: &mockSubmitEvent{}, GetEvent: &mockGetEvent{returnEvent: evt}, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/evt-1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["id"] != "evt-1" {
		t.Fatalf("expected id evt-1, got %v", body["id"])
	}
	if body["type"] != "UserRegistered" {
		t.Fatalf("expected type UserRegistered, got %v", body["type"])
	}
}

func TestGetEventHandlerNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := Handler{
		SubmitEvent: &mockSubmitEvent{},
		GetEvent:    &mockGetEvent{returnErr: apperror.NotFound("event not found", nil)},
		Logger:      zap.NewNop(),
	}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/missing", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
	assertErrorCode(t, resp, "not_found")
}

func TestGetEventHandlerUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := Handler{SubmitEvent: &mockSubmitEvent{}, GetEvent: &mockGetEvent{}, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/evt-1", nil)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestEventsSubmittedMetricSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockSubmitEvent{returnID: "evt-1"}
	handler := Handler{SubmitEvent: mock, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	// Reset counter state by reading current value before the request
	before := testutil.ToFloat64(eventsSubmittedTotal.WithLabelValues("UserRegistered", "success"))

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91b")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.Code)
	}
	after := testutil.ToFloat64(eventsSubmittedTotal.WithLabelValues("UserRegistered", "success"))
	if after-before != 1 {
		t.Fatalf("expected events_submitted_total to increment by 1, got %v", after-before)
	}
}

func TestEventsSubmittedMetricError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockSubmitEvent{returnErr: apperror.InvalidArgument("bad event", nil)}
	handler := Handler{SubmitEvent: mock, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	before := testutil.ToFloat64(eventsSubmittedTotal.WithLabelValues("UserRegistered", "error"))

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91c")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
	after := testutil.ToFloat64(eventsSubmittedTotal.WithLabelValues("UserRegistered", "error"))
	if after-before != 1 {
		t.Fatalf("expected events_submitted_total{error} to increment by 1, got %v", after-before)
	}
}
