package httpadapter

import "github.com/gin-gonic/gin"

func NewRouter(handler Handler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.POST("/api/events", handler.SubmitEventHandler)
	return router
}
