package httpadapter

import (
	"errors"
	"strconv"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	domainports "github.com/rachelJG/event-notification-service/internal/domain/ports"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"

	_ "github.com/rachelJG/event-notification-service/docs" // swagger docs
)

func NewRouter(handler Handler, adminHandler AdminHandler, health HealthChecker, apiKeyRepo domainports.APIKeyRepository, logger *zap.Logger, opts RouterOptions) *gin.Engine {
	router := gin.New()
	if err := router.SetTrustedProxies(opts.TrustedProxies); err != nil && logger != nil {
		logger.Warn("failed to set trusted proxies", zap.Error(err))
	}
	router.Use(ginzap.Ginzap(logger, "", true))
	router.Use(ginzap.RecoveryWithZap(logger, true))
	router.Use(middleware.RequestID())
	router.Use(otelgin.Middleware(opts.OTelServiceName))
	router.Use(middleware.ErrorHandler(logger))
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

	// Unauthenticated endpoints (health, metrics).
	// Rate limiting should be handled at the load balancer / API gateway level.
	publicGroup := router.Group("/")

	// Liveness: only checks that the process is running — no external I/O.
	// Used by Kubernetes livenessProbe; a failure here triggers a pod restart.
	publicGroup.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Readiness: checks that the service can handle traffic (DB reachable).
	// Used by Kubernetes readinessProbe; a failure removes the pod from rotation
	// without restarting it.
	publicGroup.GET("/health/ready", func(c *gin.Context) {
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
	publicGroup.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger API documentation
	publicGroup.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Authenticated event routes.
	v1 := router.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	})

	eventsWrite := middleware.APIKeyAuth(middleware.APIKeyOptions{
		Repo:           apiKeyRepo,
		RequiredScopes: []string{"events:write"},
		Logger:         logger,
	})
	eventsRead := middleware.APIKeyAuth(middleware.APIKeyOptions{
		Repo:           apiKeyRepo,
		RequiredScopes: []string{"events:read"},
		Logger:         logger,
	})
	v1.POST("/events", eventsWrite, handler.SubmitEventHandler)
	v1.GET("/events/:id", eventsRead, handler.GetEventHandler)

	// Admin routes — require API key with "admin" scope.
	admin := router.Group("/admin")
	admin.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	})
	adminAuth := middleware.APIKeyAuth(middleware.APIKeyOptions{
		Repo:           apiKeyRepo,
		RequiredScopes: []string{"admin"},
		Logger:         logger,
	})
	admin.POST("/api-keys", adminAuth, adminHandler.CreateAPIKeyHandler)
	admin.GET("/api-keys", adminAuth, adminHandler.ListAPIKeysHandler)
	admin.DELETE("/api-keys/:id", adminAuth, adminHandler.RevokeAPIKeyHandler)

	return router
}
