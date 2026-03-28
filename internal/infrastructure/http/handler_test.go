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

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/rachelJG/event-notification-service/internal/application/validation"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/middleware"
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

func (m *mockEventService) SubmitEvent(ctx context.Context, eventType string, payload []byte, notifications []validation.NotificationSpec, idempotencyKey, clientID string) (string, error) {
	m.submitCalled = true
	m.submitReceivedType = eventType
	m.submitReceivedKey = idempotencyKey
	return m.submitReturnID, m.submitReturnErr
}

func (m *mockEventService) GetEvent(ctx context.Context, id string) (entities.Event, error) {
	return m.getReturnEvent, m.getReturnErr
}

// fakeAPIKeyRepo is an in-memory fake for APIKeyRepository used in handler tests.
type fakeAPIKeyRepo struct {
	keys map[string]entities.APIKey // keyed by key_hash
}

func newFakeAPIKeyRepo(keys ...entities.APIKey) *fakeAPIKeyRepo {
	repo := &fakeAPIKeyRepo{keys: make(map[string]entities.APIKey)}
	for _, k := range keys {
		repo.keys[k.KeyHash] = k
	}
	return repo
}

func (r *fakeAPIKeyRepo) Create(_ context.Context, key entities.APIKey) error {
	r.keys[key.KeyHash] = key
	return nil
}

func (r *fakeAPIKeyRepo) GetByHash(_ context.Context, keyHash string) (entities.APIKey, error) {
	k, ok := r.keys[keyHash]
	if !ok {
		return entities.APIKey{}, apperror.NotFound("key not found", nil)
	}
	return k, nil
}

func (r *fakeAPIKeyRepo) List(_ context.Context) ([]entities.APIKey, error) {
	out := make([]entities.APIKey, 0, len(r.keys))
	for _, k := range r.keys {
		out = append(out, k)
	}
	return out, nil
}

func (r *fakeAPIKeyRepo) Revoke(_ context.Context, id string) error {
	for h, k := range r.keys {
		if k.ID == id {
			k.IsActive = false
			r.keys[h] = k
			return nil
		}
	}
	return apperror.NotFound("key not found", nil)
}

func (r *fakeAPIKeyRepo) UpdateLastUsed(_ context.Context, _ string) error { return nil }

// testAPIKey is the raw key used in tests. Its SHA-256 hash is stored in the fake repo.
const testRawAPIKey = "test-api-key-for-handler-tests"

// testAPIKeyEntity returns an active APIKey entity with events:write and events:read scopes.
func testAPIKeyEntity() entities.APIKey {
	return entities.APIKey{
		ID:       "key-1",
		KeyHash:  middleware.HashAPIKey(testRawAPIKey),
		Name:     "test-key",
		Scopes:   []string{"events:write", "events:read"},
		Metadata: map[string]string{"client_id": "test-client"},
		IsActive: true,
	}
}

func testRouter(handler Handler, health HealthChecker) *gin.Engine {
	repo := newFakeAPIKeyRepo(testAPIKeyEntity())
	return NewRouter(handler, AdminHandler{}, health, repo, zap.NewNop(), testRouterOptions())
}

func testRouterWithOpts(handler Handler, health HealthChecker, opts RouterOptions) *gin.Engine {
	repo := newFakeAPIKeyRepo(testAPIKeyEntity())
	return NewRouter(handler, AdminHandler{}, health, repo, zap.NewNop(), opts)
}

func TestSubmitEventHandlerMissingIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1"},"notifications":[{"channel":"email","subject":"Welcome","body":"Hello","recipients":["a@b.com"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
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
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1"},"notifications":[{"channel":"email","subject":"Welcome","body":"Hello","recipients":["a@b.com"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "not-a-uuid")
	req.Header.Set("X-API-Key", testRawAPIKey)
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
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1"},"notifications":[{"channel":"email","subject":"Welcome","body":"Hello","recipients":["a@b.com"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("X-API-Key", testRawAPIKey)
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
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1"},"notifications":[{"channel":"email","subject":"Welcome","body":"Hello","recipients":["a@b.com"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestSubmitEventUnauthorizedWithoutAuthHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1"},"notifications":[{"channel":"email","subject":"Welcome","body":"Hello","recipients":["a@b.com"]}]}`
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
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1"},"notifications":[{"channel":"email","subject":"Welcome","body":"Hello","recipients":["a@b.com"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", resp.Code)
	}
	if mock.submitCalled {
		t.Fatalf("expected use case not to be called")
	}
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
		MaxBodyBytes:        1 << 20,
		CORSAllowAllOrigins: true,
		CORSAllowedHeaders:  []string{"Origin", "Content-Length", "Content-Type", "Idempotency-Key", "X-API-Key"},
		ShutdownCh:          make(chan struct{}),
	}
}

func TestSubmitEventHandlerLocationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-abc"}
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1"},"notifications":[{"channel":"email","subject":"Welcome","body":"Hello","recipients":["a@b.com"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "7c9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("X-API-Key", testRawAPIKey)
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
	evtID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	evt := entities.Event{ID: evtID, Type: "UserRegistered", Payload: []byte(`{"user_id":"1"}`)}
	handler := Handler{EventService: &mockEventService{getReturnEvent: evt}}
	router := testRouter(handler, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+evtID, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["id"] != evtID {
		t.Fatalf("expected id %s, got %v", evtID, body["id"])
	}
	if body["type"] != "UserRegistered" {
		t.Fatalf("expected type UserRegistered, got %v", body["type"])
	}
}

func TestGetEventHandlerNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := Handler{
		EventService: &mockEventService{getReturnErr: apperror.NotFound("event not found", nil)},
	}
	router := testRouter(handler, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/b1c2d3e4-f5a6-7890-bcde-f12345678901", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
	assertErrorCode(t, resp, "not_found")
}

func TestGetEventHandlerInvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := Handler{EventService: &mockEventService{}}
	router := testRouter(handler, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/not-a-valid-uuid", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
	assertErrorCode(t, resp, "invalid_argument")
}

func TestGetEventHandlerUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := Handler{EventService: &mockEventService{}}
	router := testRouter(handler, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/c1d2e3f4-a5b6-7890-cdef-123456789012", nil)
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
	router := testRouter(Handler{EventService: &mockEventService{}}, nil)
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReadinessOKWhenDBUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := testRouter(Handler{EventService: &mockEventService{}}, &mockHealthChecker{})
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
	router := testRouter(
		Handler{EventService: &mockEventService{}},
		&mockHealthChecker{pingErr: errors.New("connection refused")},
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
	router := testRouter(Handler{EventService: &mockEventService{}}, nil)
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
	router := testRouterWithOpts(
		Handler{EventService: &mockEventService{}},
		&mockHealthChecker{}, opts,
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

func TestNewRouterDefaultBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opts := testRouterOptions()
	opts.MaxBodyBytes = 0 // triggers default fallback
	router := testRouterWithOpts(Handler{EventService: &mockEventService{}}, nil, opts)
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}



func TestEventsSubmittedMetricSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	before := testutil.ToFloat64(eventsSubmittedTotal.WithLabelValues("UserRegistered", "success"))

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1"},"notifications":[{"channel":"email","subject":"Welcome","body":"Hello","recipients":["a@b.com"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91b")
	req.Header.Set("X-API-Key", testRawAPIKey)
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
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	before := testutil.ToFloat64(httpErrorsTotal.WithLabelValues("invalid_argument"))

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1"},"notifications":[{"channel":"email","subject":"Welcome","body":"Hello","recipients":["a@b.com"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91d")
	req.Header.Set("X-API-Key", testRawAPIKey)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	after := testutil.ToFloat64(httpErrorsTotal.WithLabelValues("invalid_argument"))
	if after-before != 1 {
		t.Fatalf("expected http_errors_total{invalid_argument} to increment by 1, got %v", after-before)
	}
}


func TestSubmitEventHandlerInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnID: "evt-1"}
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(`{not valid json`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91a")
	req.Header.Set("X-API-Key", testRawAPIKey)
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
	router := testRouterWithOpts(Handler{EventService: &mockEventService{}}, nil, opts)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
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
	router := testRouterWithOpts(Handler{EventService: &mockEventService{}}, nil, opts)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Errorf("expected no HSTS header without TLS, got %q", hsts)
	}
}

func TestNewRouterInvalidTrustedProxiesLogsWarning(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opts := testRouterOptions()
	opts.TrustedProxies = []string{"not-a-valid-cidr"}
	repo := newFakeAPIKeyRepo(testAPIKeyEntity())
	router := NewRouter(Handler{EventService: &mockEventService{}}, AdminHandler{}, nil, repo, zap.NewNop(), opts)
	if router == nil {
		t.Fatal("expected non-nil router even with invalid trusted proxies")
	}
}

func TestNewRouterInvalidTrustedProxiesNilLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	opts := testRouterOptions()
	opts.TrustedProxies = []string{"not-a-valid-cidr"}
	repo := newFakeAPIKeyRepo(testAPIKeyEntity())
	router := NewRouter(Handler{EventService: &mockEventService{}}, AdminHandler{}, nil, repo, nil, opts)
	if router == nil {
		t.Fatal("expected non-nil router even with nil logger")
	}
}


func TestEventsSubmittedMetricError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockEventService{submitReturnErr: apperror.InvalidArgument("bad event", nil)}
	handler := Handler{EventService: mock}
	router := testRouter(handler, nil)

	before := testutil.ToFloat64(eventsSubmittedTotal.WithLabelValues("UserRegistered", "error"))

	reqBody := `{"event_type":"UserRegistered","payload":{"user_id":"1"},"notifications":[{"channel":"email","subject":"Welcome","body":"Hello","recipients":["a@b.com"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "6b9a1f90-6b71-4f0a-9a3d-4b72e4d9e91c")
	req.Header.Set("X-API-Key", testRawAPIKey)
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
