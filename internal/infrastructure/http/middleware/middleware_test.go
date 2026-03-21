package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// BodyLimit tests

func TestBodyLimitSetsMaxBytesReader(t *testing.T) {
	r := gin.New()
	r.POST("/test", BodyLimit(10), func(c *gin.Context) {
		buf := make([]byte, 100)
		_, err := c.Request.Body.Read(buf)
		if err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too large")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	// Body within limit
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("short"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for small body, got %d", w.Code)
	}

	// Body exceeding limit
	req = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("this body is way too long for the limit"))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for large body, got %d", w.Code)
	}
}

func TestBodyLimitNilBody(t *testing.T) {
	r := gin.New()
	r.GET("/test", BodyLimit(10), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for nil body, got %d", w.Code)
	}
}

// RequireJSONContentType tests

func TestRequireJSONContentTypePost(t *testing.T) {
	r := gin.New()
	r.POST("/test", RequireJSONContentType(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Missing content-type
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415 for missing content-type, got %d", w.Code)
	}

	// Wrong content-type
	req = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "text/plain")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415 for text/plain, got %d", w.Code)
	}

	// Correct content-type
	req = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for application/json, got %d", w.Code)
	}

	// Content-type with charset
	req = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for application/json with charset, got %d", w.Code)
	}
}

func TestRequireJSONContentTypeGetSkipped(t *testing.T) {
	r := gin.New()
	r.GET("/test", RequireJSONContentType(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for GET, got %d", w.Code)
	}
}

func TestRequireJSONContentTypePut(t *testing.T) {
	r := gin.New()
	r.PUT("/test", RequireJSONContentType(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPut, "/test", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415 for PUT without json content-type, got %d", w.Code)
	}
}

func TestRequireJSONContentTypePatch(t *testing.T) {
	r := gin.New()
	r.PATCH("/test", RequireJSONContentType(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPatch, "/test", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415 for PATCH without json content-type, got %d", w.Code)
	}
}

// RequestID tests

func TestRequestIDGeneratesNew(t *testing.T) {
	r := gin.New()
	r.GET("/test", RequestID(), func(c *gin.Context) {
		reqID := c.GetString("request_id")
		c.String(http.StatusOK, reqID)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if body == "" {
		t.Error("expected generated request ID in body")
	}
	headerID := w.Header().Get("X-Request-Id")
	if headerID == "" {
		t.Error("expected X-Request-Id header")
	}
	if body != headerID {
		t.Errorf("context ID %q != header ID %q", body, headerID)
	}
}

func TestRequestIDUsesExisting(t *testing.T) {
	r := gin.New()
	r.GET("/test", RequestID(), func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("request_id"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-Id", "custom-id-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "custom-id-123" {
		t.Errorf("expected custom-id-123, got %q", w.Body.String())
	}
	if w.Header().Get("X-Request-Id") != "custom-id-123" {
		t.Errorf("expected X-Request-Id header to be custom-id-123")
	}
}


func TestMetricsMiddlewareRecordsStatus(t *testing.T) {
	r := gin.New()
	r.GET("/test-metrics", Metrics(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
