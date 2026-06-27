package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterModelConfigRoutes(api *gin.RouterGroup, modelConfig handler.ModelConfigHandler, authMiddleware gin.HandlerFunc) {
	modelConfigRoutes := api.Group("/model-configs", authMiddleware)
	// routegen: auth user_id input=request.ListModelsRequest output=response.ListModelsResponse
	modelConfigRoutes.GET("/list", modelConfig.ListModels)
	// routegen: auth user_id input=request.CreateModelRequest output=response.ModelResponse
	modelConfigRoutes.POST("/", modelConfig.CreateModel)
	modelConfigRoutes.POST("/create", modelConfig.CreateModel)
	// routegen: auth user_id input=request.UpdateModelRequest output=response.ModelResponse
	modelConfigRoutes.PATCH("/:config_id", modelConfig.UpdateModel)
	// routegen: auth user_id input=request.UriConfigIDRequest
	modelConfigRoutes.DELETE("/:config_id", modelConfig.DeleteModel)
	// routegen: auth user_id input=request.UriConfigIDRequest output=response.ModelResponse
	modelConfigRoutes.POST("/:config_id", modelConfig.SetModelDefault)
}
