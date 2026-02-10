package httpadapter

import "github.com/gin-gonic/gin"

func NewRouter(handler Handler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	v1 := router.Group("/api/v1")
	v1.POST("/events", handler.SubmitEventHandler)
	return router
}
