package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterModelConfigRoutes(api *gin.RouterGroup, modelConfig handler.ModelConfigHandler, authMiddleware gin.HandlerFunc) {
	modelConfigRoutes := api.Group("/model-configs", authMiddleware)
	modelConfigRoutes.GET("", modelConfig.List)
	modelConfigRoutes.POST("", modelConfig.Create)
}
