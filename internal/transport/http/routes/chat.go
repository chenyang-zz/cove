package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterChatRoutes(api *gin.RouterGroup, chat handler.ChatHandler, authMiddleware gin.HandlerFunc) {
	chatRoutes := api.Group("/chat", authMiddleware)
	chatRoutes.POST("/stream", chat.Stream)
}
