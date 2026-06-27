package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterHealthRoutes(api *gin.RouterGroup, health handler.HealthHandler) {
	// routegen: output=response.HealthResponse
	api.GET("/health", health.Health)
}
