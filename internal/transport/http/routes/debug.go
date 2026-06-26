package routes

import "github.com/gin-gonic/gin"

func RegisterDebugRoutes(api *gin.RouterGroup) {
	debugRoutes := api.Group("/debug")
	debugRoutes.GET("/panic", func(c *gin.Context) {
		panic("debug panic")
	})
}
