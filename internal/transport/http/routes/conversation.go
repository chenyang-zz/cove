/**
 * @Time   : 2026/6/27 15:38
 * @Author : chenyangzhao542@gmail.com
 * @File   : conversation.go
 **/

package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterConversationRoutes(api *gin.RouterGroup, conversation handler.ConversationHandler, authMiddleware gin.HandlerFunc) {
	conversationRoutes := api.Group("/conversation", authMiddleware)
	// routegen: auth user_id input=request.CreateConversationRequest output=response.ConversationResponse
	conversationRoutes.POST("/", conversation.CreateConversation)
	// routegen: auth user_id output=response.ListResponse[*response.ConversationResponse]
	conversationRoutes.GET("/", conversation.ListConversations)
	// @auth(user_id)
	// @description 重命名会话
	// @input request.RenameConversationRequest
	// @response ConversationResponse
	conversationRoutes.PATCH("/:conversation_id", conversation.RenameConversation)
	conversationRoutes.POST("/:conversation_id/rename", conversation.RenameConversation)
	// @auth(user_id)
	// @description 删除会话
	// @input request.UriConversationIDRequest
	conversationRoutes.DELETE("/:conversation_id", conversation.DeleteConversation)
	conversationRoutes.POST("/:conversation_id/delete", conversation.DeleteConversation)
	// @auth(user_id)
	// @description 获取消息列表
	// @input request.UriConversationIDRequest
	// @response ListResponse[*response.MessageResponse]
	conversationRoutes.GET("/:conversation_id/messages", conversation.ListMessages)
}
