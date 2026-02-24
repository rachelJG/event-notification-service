package httpadapter

import (
	"errors"
	"strconv"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/middleware"
	"go.uber.org/zap"
)

func NewRouter(handler Handler, health HealthChecker, logger *zap.Logger, opts RouterOptions) *gin.Engine {
	router := gin.New()
	if err := router.SetTrustedProxies(opts.TrustedProxies); err != nil && logger != nil {
		logger.Warn("failed to set trusted proxies", zap.Error(err))
	}
	router.Use(ginzap.Ginzap(logger, "", true))
	router.Use(ginzap.RecoveryWithZap(logger, true))
	router.Use(middleware.RequestID())
	// Security Headers
	router.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		if opts.EnableHSTS && c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age="+strconv.Itoa(opts.HSTSMaxAgeSeconds)+"; includeSubDomains")
		}
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	})

	// Observability
	router.Use(middleware.Metrics())

	// CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = opts.CORSAllowAllOrigins
	corsConfig.AllowOrigins = opts.CORSAllowedOrigins
	corsConfig.AllowHeaders = opts.CORSAllowedHeaders
	router.Use(cors.New(corsConfig))

	bodyLimit := opts.MaxBodyBytes
	if bodyLimit <= 0 {
		bodyLimit = 1 << 20
	}
	router.Use(middleware.BodyLimit(bodyLimit))
	router.Use(middleware.RequireJSONContentType())
	rps := opts.RateLimitRPS
	if rps <= 0 {
		rps = 10
	}
	burst := opts.RateLimitBurst
	if burst <= 0 {
		burst = 20
	}
	router.Use(middleware.RateLimit(rps, burst))

	// Liveness: only checks that the process is running — no external I/O.
	// Used by Kubernetes livenessProbe; a failure here triggers a pod restart.
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Readiness: checks that the service can handle traffic (DB reachable).
	// Used by Kubernetes readinessProbe; a failure removes the pod from rotation
	// without restarting it.
	router.GET("/health/ready", func(c *gin.Context) {
		if health == nil {
			if logger != nil {
				logger.Error("readiness check failed", zap.Error(errors.New("health checker is nil")))
			}
			c.JSON(503, gin.H{"error": "database unavailable", "code": "internal", "request_id": c.GetString("request_id")})
			return
		}
		if err := health.Ping(c.Request.Context()); err != nil {
			if logger != nil {
				logger.Error("readiness check failed", zap.Error(err))
			}
			c.JSON(503, gin.H{"error": "database unavailable", "code": "internal", "request_id": c.GetString("request_id")})
			return
		}
		c.JSON(200, gin.H{
			"status":  "ok",
			"version": opts.Version,
			"commit":  opts.Commit,
			"db":      health.Stats(),
		})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	})
	jwtAuth := middleware.JWTAuth(middleware.JWTOptions{
		Secret:   opts.JWTSecret,
		Issuer:   opts.JWTIssuer,
		Audience: opts.JWTAudience,
		Logger:   logger,
	})
	v1.POST("/events", jwtAuth, handler.SubmitEventHandler)
	v1.GET("/events/:id", jwtAuth, handler.GetEventHandler)
	return router
}
