package httpadapter

import (
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/rachelJG/event-notification-service/internal/adapters/http/middleware"
	"github.com/rachelJG/event-notification-service/internal/config"
	"go.uber.org/zap"
)

func NewRouter(handler Handler, logger *zap.Logger, cfg config.Config) *gin.Engine {
	router := gin.New()
	router.Use(ginzap.Ginzap(logger, "", true))
	router.Use(ginzap.RecoveryWithZap(logger, true))
	router.Use(middleware.RequestID())
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
		c.JSON(200, gin.H{"status": "ok"})
	})
	v1 := router.Group("/api/v1")
	v1.POST("/events", middleware.JWTAuth(cfg.JWTSecret), handler.SubmitEventHandler)
	return router
}
