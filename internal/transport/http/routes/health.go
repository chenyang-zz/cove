package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterHealthRoutes(api *gin.RouterGroup, health handler.HealthHandler) {
	api.GET("/hello", health.Hello)
	api.GET("/health", health.Health)
}
