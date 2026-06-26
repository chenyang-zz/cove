package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(api *gin.RouterGroup, auth handler.AuthHandler, authMiddleware gin.HandlerFunc) {
	authRoutes := api.Group("/auth")
	authRoutes.POST("/register", auth.Register)
	authRoutes.POST("/login", auth.Login)
	authRoutes.POST("/refresh", auth.Refresh)

	api.GET("/me", authMiddleware, auth.Me)
}
