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

	router.GET("/health", func(c *gin.Context) {
		if health == nil {
			if logger != nil {
				logger.Error("health check failed", zap.Error(errors.New("health checker is nil")))
			}
			c.JSON(503, gin.H{"error": "database unavailable", "code": "internal"})
			return
		}
		if err := health.Ping(c.Request.Context()); err != nil {
			if logger != nil {
				logger.Error("health check failed", zap.Error(err))
			}
			c.JSON(503, gin.H{"error": "database unavailable", "code": "internal"})
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
	jwtAuth := middleware.JWTAuth(middleware.JWTOptions{
		Secret:   opts.JWTSecret,
		Issuer:   opts.JWTIssuer,
		Audience: opts.JWTAudience,
	})
	v1.POST("/events", jwtAuth, handler.SubmitEventHandler)
	v1.GET("/events/:id", jwtAuth, handler.GetEventHandler)
	return router
}
