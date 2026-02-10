package httpadapter

import (
	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/rachelJG/event-notification-service/internal/adapters/http/middleware"
	"go.uber.org/zap"
)

func NewRouter(handler Handler, logger *zap.Logger, jwtSecret string) *gin.Engine {
	router := gin.New()
	router.Use(ginzap.Ginzap(logger, "", true))
	router.Use(ginzap.RecoveryWithZap(logger, true))
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	v1 := router.Group("/api/v1")
	v1.POST("/events", middleware.JWTAuth(jwtSecret), handler.SubmitEventHandler)
	return router
}
