package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimit(requestsPerSecond float64, burst int) gin.HandlerFunc {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*ipLimiter)
	)

	// cleanup goroutine to avoid unbounded growth
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 10*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		v, ok := visitors[ip]
		if !ok {
			limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)
			visitors[ip] = &ipLimiter{limiter: limiter, lastSeen: time.Now()}
			return limiter
		}
		v.lastSeen = time.Now()
		return v.limiter
	}

	return func(c *gin.Context) {
		ip := clientIP(c.Request)
		if ip == "" {
			ip = c.ClientIP()
		}
		limiter := getLimiter(ip)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return host
}
