package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// RateLimit (IP-based)
// ---------------------------------------------------------------------------

func TestRateLimit_AllowsWithinBurst(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		rps   float64
		burst int
		count int
	}{
		{name: "burst_1", rps: 100, burst: 1, count: 1},
		{name: "burst_5", rps: 100, burst: 5, count: 5},
		{name: "burst_10", rps: 100, burst: 10, count: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := gin.New()
			r.GET("/test", RateLimit(tt.rps, tt.burst, testDone()), func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			for i := range tt.count {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = "1.2.3.4:1234"
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed within burst", i)
			}
		})
	}
}

func TestRateLimit_ExceedsBurst(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test", RateLimit(1, 1, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// First request succeeds (consumes the single burst token)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "5.6.7.8:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "first request should succeed")

	// Second request should be rate limited
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "5.6.7.8:1234"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "second request should be rate limited")
	assert.NotEmpty(t, w.Header().Get("Retry-After"), "should include Retry-After header")
}

func TestRateLimit_PerIPIsolation(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test", RateLimit(1, 1, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	tests := []struct {
		name       string
		ip         string
		wantStatus int
	}{
		{name: "ip1_first_request", ip: "10.0.0.1:1234", wantStatus: http.StatusOK},
		{name: "ip1_rate_limited", ip: "10.0.0.1:1234", wantStatus: http.StatusTooManyRequests},
		{name: "ip2_separate_limiter", ip: "10.0.0.2:1234", wantStatus: http.StatusOK},
		{name: "ip2_rate_limited", ip: "10.0.0.2:1234", wantStatus: http.StatusTooManyRequests},
		{name: "ip3_separate_limiter", ip: "10.0.0.3:1234", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.ip
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestRateLimit_ResponseBody(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test", RequestID(), RateLimit(1, 1, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Exhaust burst
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.50:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Trigger rate limit and inspect response
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.50:1234"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusTooManyRequests, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err, "response body should be valid JSON")

	assert.Equal(t, "rate limit exceeded", body["error"], "should contain error message")
	assert.Equal(t, "rate_limited", body["code"], "should contain error code")
	assert.NotEmpty(t, w.Header().Get("Retry-After"), "should include Retry-After header")
}

func TestRateLimit_RetryAfterValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		rps            float64
		wantRetryAfter string
	}{
		{name: "1_rps", rps: 1, wantRetryAfter: "1"},
		{name: "0.5_rps", rps: 0.5, wantRetryAfter: "2"},
		{name: "0.1_rps", rps: 0.1, wantRetryAfter: "10"},
		{name: "10_rps", rps: 10, wantRetryAfter: "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := gin.New()
			r.GET("/test", RateLimit(tt.rps, 1, testDone()), func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			// Exhaust burst
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "99.99.99.99:1234"
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			// Trigger 429
			req = httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "99.99.99.99:1234"
			w = httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusTooManyRequests, w.Code)
			assert.Equal(t, tt.wantRetryAfter, w.Header().Get("Retry-After"))
		})
	}
}

func TestRateLimit_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test", RateLimit(1000, 50, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			// We don't assert specific status here — just ensuring no race/panic.
			assert.Contains(t, []int{http.StatusOK, http.StatusTooManyRequests}, w.Code,
				"goroutine %d: unexpected status", i)
		}(i)
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// APIKeyRateLimit (API-key-based)
// ---------------------------------------------------------------------------

func TestAPIKeyRateLimit_KeysByAPIKeyName(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test", setAPIKeyName("service-a"), APIKeyRateLimit(1, 1, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// First request succeeds
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "first request should succeed")

	// Second request from same API key is rate limited — even from a different IP
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code,
		"same API key from different IP should still be rate limited")
}

func TestAPIKeyRateLimit_IsolationBetweenKeys(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test/:key", func(c *gin.Context) {
		c.Set(ContextKeyAPIKeyName, c.Param("key"))
		c.Next()
	}, APIKeyRateLimit(1, 1, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	tests := []struct {
		name       string
		key        string
		wantStatus int
	}{
		{name: "service-a_first", key: "service-a", wantStatus: http.StatusOK},
		{name: "service-a_rate_limited", key: "service-a", wantStatus: http.StatusTooManyRequests},
		{name: "service-b_independent", key: "service-b", wantStatus: http.StatusOK},
		{name: "service-b_rate_limited", key: "service-b", wantStatus: http.StatusTooManyRequests},
		{name: "service-c_independent", key: "service-c", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test/"+tt.key, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestAPIKeyRateLimit_FallsBackToIPWhenNoKey(t *testing.T) {
	t.Parallel()

	r := gin.New()
	// No API key name set in context — should fall back to IP-based keying.
	r.GET("/test", APIKeyRateLimit(1, 1, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	tests := []struct {
		name       string
		ip         string
		wantStatus int
	}{
		{name: "ip1_first", ip: "10.0.0.1:1234", wantStatus: http.StatusOK},
		{name: "ip1_limited", ip: "10.0.0.1:1234", wantStatus: http.StatusTooManyRequests},
		{name: "ip2_independent", ip: "10.0.0.2:1234", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.ip
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestAPIKeyRateLimit_SameIPDifferentKeys(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test/:key", func(c *gin.Context) {
		c.Set(ContextKeyAPIKeyName, c.Param("key"))
		c.Next()
	}, APIKeyRateLimit(1, 1, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Both requests come from the same IP but with different API keys.
	// They should have independent rate limits.
	req := httptest.NewRequest(http.MethodGet, "/test/key-alpha", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "key-alpha first request should succeed")

	req = httptest.NewRequest(http.MethodGet, "/test/key-beta", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code,
		"key-beta should succeed even though same IP as key-alpha")
}

func TestAPIKeyRateLimit_AllowsWithinBurst(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test", setAPIKeyName("burst-service"), APIKeyRateLimit(100, 5, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	for i := range 5 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed within burst", i)
	}
}

func TestAPIKeyRateLimit_ResponseBody(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test", RequestID(), setAPIKeyName("resp-svc"), APIKeyRateLimit(1, 1, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Exhaust burst
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Trigger rate limit
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusTooManyRequests, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err, "response should be valid JSON")

	assert.Equal(t, "rate limit exceeded", body["error"])
	assert.Equal(t, "rate_limited", body["code"])
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestAPIKeyRateLimit_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	r := gin.New()
	r.GET("/test", setAPIKeyName("concurrent-svc"), APIKeyRateLimit(1000, 50, testDone()), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Contains(t, []int{http.StatusOK, http.StatusTooManyRequests}, w.Code,
				"goroutine %d: unexpected status", i)
		}(i)
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// testDone returns a done channel that is never closed, suitable for tests
// where cleanup goroutine lifecycle is irrelevant.
func testDone() <-chan struct{} {
	return make(chan struct{})
}

// setAPIKeyName returns a middleware that sets the API key name in gin context,
// simulating APIKeyAuth having already authenticated the request.
func setAPIKeyName(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ContextKeyAPIKeyName, name)
		c.Next()
	}
}
