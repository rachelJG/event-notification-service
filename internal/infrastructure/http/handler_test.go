package httpadapter

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
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

type mockEventService struct {
	submitCalled       bool
	submitReturnID     string
	submitReturnErr    error
	submitReceivedType string
	submitReceivedKey  string
	getReturnEvent     entities.Event
	getReturnErr       error
}

func (m *mockEventService) SubmitEvent(ctx context.Context, eventType string, payload []byte, idempotencyKey string) (string, error) {
	m.submitCalled = true
	m.submitReceivedType = eventType
	m.submitReceivedKey = idempotencyKey
	return m.submitReturnID, m.submitReturnErr
}

func (m *mockEventService) GetEvent(ctx context.Context, id string) (entities.Event, error) {
	return m.getReturnEvent, m.getReturnErr
}

func TestSubmitEventHandlerMissingIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
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
	if mock.submitCalled {
		t.Fatalf("expected use case not to be called")
	}
	assertErrorCode(t, resp, "invalid_argument")
}

func TestSubmitEventHandlerInvalidIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
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
	if mock.submitCalled {
		t.Fatalf("expected use case not to be called")
	}
	assertErrorCode(t, resp, "invalid_argument")
}

func TestSubmitEventHandlerSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
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
	if !mock.submitCalled {
		t.Fatalf("expected use case to be called")
	}
	if mock.submitReceivedKey != "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a" {
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
	mock := &mockEventService{submitReturnErr: apperror.InvalidArgument("bad event", nil)}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
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
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
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
	if mock.submitCalled {
		t.Fatalf("expected use case not to be called")
	}
}

func TestSubmitEventRejectsNonJSONContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
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
	if mock.submitCalled {
		t.Fatalf("expected use case not to be called")
	}
}

func TestSubmitEventRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
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
	if body["request_id"] == "" {
		t.Fatalf("expected request_id in error response, got empty")
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
	mock := &mockEventService{submitReturnID: "evt-abc"}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
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
	handler := Handler{EventService: &mockEventService{getReturnEvent: evt}, Logger: zap.NewNop()}
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
		EventService: &mockEventService{getReturnErr: apperror.NotFound("event not found", nil)},
		Logger:       zap.NewNop(),
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
	handler := Handler{EventService: &mockEventService{}, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/evt-1", nil)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

// --- health probe tests ---

type mockHealthChecker struct {
	pingErr error
}

func (m *mockHealthChecker) Ping(_ context.Context) error { return m.pingErr }
func (m *mockHealthChecker) Stats() DBStats               { return DBStats{MaxConns: 10} }

func TestLivenessAlwaysOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// health=nil: liveness must not touch the DB
	router := NewRouter(Handler{EventService: &mockEventService{}}, nil, zap.NewNop(), testRouterOptions())
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReadinessOKWhenDBUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(
		Handler{EventService: &mockEventService{}},
		&mockHealthChecker{}, zap.NewNop(), testRouterOptions(),
	)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", body["status"])
	}
}

func TestReadiness503WhenDBDown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(
		Handler{EventService: &mockEventService{}},
		&mockHealthChecker{pingErr: errors.New("connection refused")}, zap.NewNop(), testRouterOptions(),
	)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestReadiness503WhenHealthCheckerNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(
		Handler{EventService: &mockEventService{}},
		nil, zap.NewNop(), testRouterOptions(),
	)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestReadinessReturnsVersionAndDBStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opts := testRouterOptions()
	opts.Version = "1.0.0"
	opts.Commit = "abc123"
	router := NewRouter(
		Handler{EventService: &mockEventService{}},
		&mockHealthChecker{}, zap.NewNop(), opts,
	)
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["version"] != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %v", body["version"])
	}
	if body["commit"] != "abc123" {
		t.Errorf("expected commit abc123, got %v", body["commit"])
	}
	db, ok := body["db"].(map[string]any)
	if !ok {
		t.Fatalf("expected db stats object in response")
	}
	if db["max_conns"] != float64(10) {
		t.Errorf("expected max_conns 10, got %v", db["max_conns"])
	}
}

func TestRequestIDFromContextEdgeCases(t *testing.T) {
	// Test requestIDFromContext when key is not a string
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set("request_id", 12345) // not a string
		result := requestIDFromContext(c)
		if result != "" {
			t.Errorf("expected empty string for non-string request_id, got %q", result)
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
}

func TestRequestIDFromContextMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		result := requestIDFromContext(c)
		if result != "" {
			t.Errorf("expected empty string for missing request_id, got %q", result)
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
}

func TestEventsSubmittedMetricSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
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

func TestHTTPErrorsMetricIncremented(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnErr: apperror.InvalidArgument("bad event", nil)}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	before := testutil.ToFloat64(httpErrorsTotal.WithLabelValues("invalid_argument"))

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91d")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	after := testutil.ToFloat64(httpErrorsTotal.WithLabelValues("invalid_argument"))
	if after-before != 1 {
		t.Fatalf("expected http_errors_total{invalid_argument} to increment by 1, got %v", after-before)
	}
}

func TestSubmitEventHandlerNilLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnErr: apperror.InvalidArgument("bad event", nil)}
	handler := Handler{EventService: mock, Logger: nil}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1","email":"a@b.com","name":"A"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e920")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestSubmitEventHandlerInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
	router := NewRouter(handler, nil, zap.NewNop(), testRouterOptions())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(`{not valid json`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("Authorization", "Bearer "+testJWT(t, "test-secret"))
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
	assertErrorCode(t, resp, "invalid_argument")
	if mock.submitCalled {
		t.Fatal("expected use case not to be called for invalid JSON")
	}
}

func TestHSTSHeaderWithTLS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opts := testRouterOptions()
	opts.EnableHSTS = true
	opts.HSTSMaxAgeSeconds = 31536000
	router := NewRouter(Handler{EventService: &mockEventService{}}, nil, zap.NewNop(), opts)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	// Simulate TLS by setting TLS field
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Fatal("expected Strict-Transport-Security header when TLS is present and HSTS enabled")
	}
	if hsts != "max-age=31536000; includeSubDomains" {
		t.Errorf("unexpected HSTS value: %q", hsts)
	}
}

func TestNoHSTSWithoutTLS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opts := testRouterOptions()
	opts.EnableHSTS = true
	opts.HSTSMaxAgeSeconds = 31536000
	router := NewRouter(Handler{EventService: &mockEventService{}}, nil, zap.NewNop(), opts)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	// No TLS
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Errorf("expected no HSTS header without TLS, got %q", hsts)
	}
}

func TestEventTypeFromContextNonString(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set("event_type", 12345)
		result := eventTypeFromContext(c)
		if result != "" {
			t.Errorf("expected empty for non-string event_type, got %q", result)
		}
		c.String(http.StatusOK, "ok")
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
}

func TestIdempotencyKeyFromContextNonString(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.Set("idempotency_key", 12345)
		result := idempotencyKeyFromContext(c)
		if result != "" {
			t.Errorf("expected empty for non-string idempotency_key, got %q", result)
		}
		c.String(http.StatusOK, "ok")
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
}

func TestEventsSubmittedMetricError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnErr: apperror.InvalidArgument("bad event", nil)}
	handler := Handler{EventService: mock, Logger: zap.NewNop()}
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
