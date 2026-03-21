package httputil

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// execHandler creates a Gin router, registers handler on GET /test, and returns the recorder.
func execHandler(t *testing.T, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	r.GET("/test", handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// parseJSON unmarshals the response body into an ErrorResponse.
func parseJSON(t *testing.T, w *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()
	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "response body should be valid JSON")
	return resp
}

func TestWriteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		requestID  string
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{
			name:       "unauthenticated maps to 401",
			err:        apperror.Unauthenticated("bad token", nil),
			requestID:  "req-1",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthenticated",
			wantMsg:    "bad token",
		},
		{
			name:       "permission_denied maps to 403",
			err:        apperror.PermissionDenied("forbidden", nil),
			wantStatus: http.StatusForbidden,
			wantCode:   "permission_denied",
			wantMsg:    "forbidden",
		},
		{
			name:       "invalid_argument maps to 400",
			err:        apperror.InvalidArgument("bad input", nil),
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_argument",
			wantMsg:    "bad input",
		},
		{
			name:       "not_found maps to 404",
			err:        apperror.NotFound("missing", nil),
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
			wantMsg:    "missing",
		},
		{
			name:       "conflict maps to 409",
			err:        apperror.Conflict("duplicate", nil),
			wantStatus: http.StatusConflict,
			wantCode:   "conflict",
			wantMsg:    "duplicate",
		},
		{
			name:       "rate_limited maps to 429",
			err:        apperror.New(apperror.CodeRateLimited, "slow down", nil),
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "rate_limited",
			wantMsg:    "slow down",
		},
		{
			name:       "internal maps to 500",
			err:        apperror.Internal("server broke", nil),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal",
			wantMsg:    "server broke",
		},
		{
			name:       "generic error defaults to 500 internal",
			err:        errors.New("something unexpected"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal",
			wantMsg:    "internal error",
		},
		{
			name:       "nil wrapped error still maps correctly",
			err:        apperror.NotFound("gone", nil),
			requestID:  "req-nil-wrap",
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
			wantMsg:    "gone",
		},
		{
			name:       "AppError with wrapped cause preserves mapping",
			err:        apperror.Internal("db failed", errors.New("connection refused")),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal",
			wantMsg:    "db failed",
		},
		{
			name:       "timeout maps to 500",
			err:        apperror.Timeout("too slow", nil),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal",
			wantMsg:    "internal error",
		},
		{
			name:       "empty request_id omitted from JSON",
			err:        apperror.NotFound("nope", nil),
			requestID:  "",
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
			wantMsg:    "nope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := execHandler(t, func(c *gin.Context) {
				if tt.requestID != "" {
					c.Set("request_id", tt.requestID)
				}
				WriteError(c, tt.err)
			})

			assert.Equal(t, tt.wantStatus, w.Code, "HTTP status")

			resp := parseJSON(t, w)
			assert.Equal(t, tt.wantMsg, resp.Error, "error message")
			assert.Equal(t, tt.wantCode, resp.Code, "error code")

			if tt.requestID != "" {
				assert.Equal(t, tt.requestID, resp.RequestID, "request_id")
			} else {
				assert.Empty(t, resp.RequestID, "request_id should be empty when not set")
			}
		})
	}
}

func TestWriteError_AbortsRequest(t *testing.T) {
	t.Parallel()

	nextCalled := false
	r := gin.New()
	r.GET("/test",
		func(c *gin.Context) {
			WriteError(c, apperror.Unauthenticated("denied", nil))
		},
		func(c *gin.Context) {
			nextCalled = true
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, nextCalled, "WriteError should abort the request chain")
}

func TestWritePermissionError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		scope     string
		requestID string
		wantMsg   string
	}{
		{
			name:      "admin scope",
			scope:     "admin",
			requestID: "req-perm-1",
			wantMsg:   "insufficient scope: admin",
		},
		{
			name:      "events:write scope",
			scope:     "events:write",
			requestID: "req-perm-2",
			wantMsg:   "insufficient scope: events:write",
		},
		{
			name:      "empty scope",
			scope:     "",
			requestID: "req-perm-3",
			wantMsg:   "insufficient scope: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := execHandler(t, func(c *gin.Context) {
				c.Set("request_id", tt.requestID)
				WritePermissionError(c, tt.scope)
			})

			assert.Equal(t, http.StatusForbidden, w.Code, "HTTP status")

			resp := parseJSON(t, w)
			assert.Equal(t, tt.wantMsg, resp.Error, "error message")
			assert.Equal(t, "permission_denied", resp.Code, "error code")
			assert.Equal(t, tt.requestID, resp.RequestID, "request_id")
		})
	}
}

func TestWriteCustomError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		message    string
		code       string
		requestID  string
		wantStatus int
	}{
		{
			name:       "teapot",
			status:     http.StatusTeapot,
			message:    "i'm a teapot",
			code:       "teapot",
			requestID:  "req-custom-1",
			wantStatus: http.StatusTeapot,
		},
		{
			name:       "service unavailable",
			status:     http.StatusServiceUnavailable,
			message:    "maintenance",
			code:       "unavailable",
			requestID:  "req-custom-2",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "no request_id",
			status:     http.StatusBadGateway,
			message:    "upstream down",
			code:       "bad_gateway",
			requestID:  "",
			wantStatus: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := execHandler(t, func(c *gin.Context) {
				if tt.requestID != "" {
					c.Set("request_id", tt.requestID)
				}
				WriteCustomError(c, tt.status, tt.message, tt.code)
			})

			assert.Equal(t, tt.wantStatus, w.Code, "HTTP status")

			resp := parseJSON(t, w)
			assert.Equal(t, tt.message, resp.Error, "error message")
			assert.Equal(t, tt.code, resp.Code, "error code")

			if tt.requestID != "" {
				assert.Equal(t, tt.requestID, resp.RequestID, "request_id")
			} else {
				assert.Empty(t, resp.RequestID, "request_id should be empty")
			}
		})
	}
}

func TestWriteCustomError_AbortsRequest(t *testing.T) {
	t.Parallel()

	nextCalled := false
	r := gin.New()
	r.GET("/test",
		func(c *gin.Context) {
			WriteCustomError(c, http.StatusForbidden, "nope", "denied")
		},
		func(c *gin.Context) {
			nextCalled = true
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, nextCalled, "WriteCustomError should abort the request chain")
}

func TestErrorResponse_JSONOmitsEmptyRequestID(t *testing.T) {
	t.Parallel()

	resp := ErrorResponse{
		Error: "test",
		Code:  "test_code",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))

	_, hasRequestID := raw["request_id"]
	assert.False(t, hasRequestID, "request_id should be omitted from JSON when empty")
}
