package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit applies IP-based rate limiting. Use this for unauthenticated
// endpoints (health, metrics) where there is no API key to identify the caller.
// The done channel controls the cleanup goroutine lifetime; close it to stop cleanup.
func RateLimit(requestsPerSecond float64, burst int, done <-chan struct{}) gin.HandlerFunc {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*visitor)
	)

	retryAfter := strconv.Itoa(int(math.Ceil(1.0 / requestsPerSecond)))

	go cleanupLoop(done, &mu, visitors)

	getLimiter := func(key string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		v, ok := visitors[key]
		if !ok {
			limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)
			visitors[key] = &visitor{limiter: limiter, lastSeen: time.Now()}
			return limiter
		}
		v.lastSeen = time.Now()
		return v.limiter
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := getLimiter(ip)
		if !limiter.Allow() {
			c.Header("Retry-After", retryAfter)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded", "code": "rate_limited", "request_id": c.GetString("request_id")})
			return
		}
		c.Next()
	}
}

// APIKeyRateLimit applies rate limiting keyed by the authenticated API key name.
// It must be placed after APIKeyAuth middleware so that ContextKeyAPIKeyName is
// set in the gin context. If no API key name is found (should not happen on
// authenticated routes), it falls back to IP-based limiting.
// The done channel controls the cleanup goroutine lifetime; close it to stop cleanup.
func APIKeyRateLimit(requestsPerSecond float64, burst int, done <-chan struct{}) gin.HandlerFunc {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*visitor)
	)

	retryAfter := strconv.Itoa(int(math.Ceil(1.0 / requestsPerSecond)))

	go cleanupLoop(done, &mu, visitors)

	getLimiter := func(key string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		v, ok := visitors[key]
		if !ok {
			limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)
			visitors[key] = &visitor{limiter: limiter, lastSeen: time.Now()}
			return limiter
		}
		v.lastSeen = time.Now()
		return v.limiter
	}

	return func(c *gin.Context) {
		key := c.GetString(ContextKeyAPIKeyName)
		if key == "" {
			key = "ip:" + c.ClientIP()
		}

		limiter := getLimiter(key)
		if !limiter.Allow() {
			c.Header("Retry-After", retryAfter)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded", "code": "rate_limited", "request_id": c.GetString("request_id")})
			return
		}
		c.Next()
	}
}

func cleanupLoop(done <-chan struct{}, mu *sync.Mutex, visitors map[string]*visitor) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			mu.Lock()
			for key, v := range visitors {
				if time.Since(v.lastSeen) > 10*time.Minute {
					delete(visitors, key)
				}
			}
			mu.Unlock()
		}
	}
}
