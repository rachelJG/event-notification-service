package httpadapter

import (
	"errors"
	"strconv"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rachelJG/event-notification-service/internal/config"
	"github.com/rachelJG/event-notification-service/internal/infrastructure/http/middleware"
	"go.uber.org/zap"
)

func NewRouter(handler Handler, db *pgxpool.Pool, logger *zap.Logger, cfg config.Config) *gin.Engine {
	router := gin.New()
	router.Use(ginzap.Ginzap(logger, "", true))
	router.Use(ginzap.RecoveryWithZap(logger, true))
	router.Use(middleware.RequestID())
	// Security Headers
	router.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		if cfg.EnableHSTS && c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age="+strconv.Itoa(cfg.HSTSMaxAgeSeconds)+"; includeSubDomains")
		}
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	})

	// Observability
	router.Use(middleware.Metrics())

	// CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = cfg.CORSAllowAllOrigins
	corsConfig.AllowOrigins = cfg.CORSAllowedOrigins
	corsConfig.AllowHeaders = cfg.CORSAllowedHeaders
	router.Use(cors.New(corsConfig))

	bodyLimit := cfg.MaxBodyBytes
	if bodyLimit <= 0 {
		bodyLimit = 1 << 20
	}
	router.Use(middleware.BodyLimit(bodyLimit))
	router.Use(middleware.RequireJSONContentType())
	rps := cfg.RateLimitRPS
	if rps <= 0 {
		rps = 10
	}
	burst := cfg.RateLimitBurst
	if burst <= 0 {
		burst = 20
	}
	router.Use(middleware.RateLimit(rps, burst))

	router.GET("/health", func(c *gin.Context) {
		if db == nil {
			if logger != nil {
				logger.Error("health check failed", zap.Error(errors.New("database pool is nil")))
			}
			c.JSON(503, gin.H{"status": "error", "message": "database unavailable"})
			return
		}
		if err := db.Ping(c.Request.Context()); err != nil {
			if logger != nil {
				logger.Error("health check failed", zap.Error(err))
			}
			c.JSON(503, gin.H{"status": "error", "message": "database unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := router.Group("/api/v1")
	v1.POST("/events", middleware.JWTAuth(cfg.JWTSecret), handler.SubmitEventHandler)
	return router
}
