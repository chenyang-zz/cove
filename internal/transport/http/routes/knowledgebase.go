/**
 * @Time   : 2026/6/30 18:06
 * @Author : chenyangzhao542@gmail.com
 * @File   : knowledgebase.go
 **/

package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterKnowledgeBaseRoutes(api *gin.RouterGroup, knowledgebase handler.KnowledgeBaseHandler, authMiddleware gin.HandlerFunc) {
	knowledgebaseRoutes := api.Group("/knowledge-base", authMiddleware)

	// @auth(user_id)
	// @description 查询知识库
	// @input request.UriKnowledgeBaseIDRequest
	// @output response.KnowledgeBaseResponse
	knowledgebaseRoutes.GET("/:k_id", knowledgebase.GetKnowledgeBase)

	// @auth(user_id)
	// @description 查询知识库列表
	// @output response.ListResponse[*response.KnowledgeBaseResponse]
	knowledgebaseRoutes.GET("/", knowledgebase.GetKnowledgeBaseList)
	knowledgebaseRoutes.GET("/list", knowledgebase.GetKnowledgeBaseList)

	// @auth(user_id)
	// @description 创建知识库
	// @input request.CreateKnowledgeBaseRequest
	// @output response.KnowledgeBaseResponse
	knowledgebaseRoutes.POST("/", knowledgebase.CreateKnowledgeBase)
	knowledgebaseRoutes.POST("/create", knowledgebase.CreateKnowledgeBase)

	// @auth(user_id)
	// @description 更新知识库
	// @input request.UpdateKnowledgeBaseRequest
	// @output response.KnowledgeBaseResponse
	knowledgebaseRoutes.PATCH("/:k_id", knowledgebase.UpdateKnowledgeBase)
	knowledgebaseRoutes.PATCH("/:k_id/update", knowledgebase.UpdateKnowledgeBase)

	// @auth(user_id)
	// @description 删除知识库
	// @input request.UriKnowledgeBaseIDRequest
	knowledgebaseRoutes.DELETE("/:k_id", knowledgebase.DeleteKnowledgeBase)
	knowledgebaseRoutes.POST("/:k_id/delete", knowledgebase.DeleteKnowledgeBase)

	// @auth(user_id)
	// @description 启用或禁用知识库聊天
	// @input request.EnabledChatRequest
	knowledgebaseRoutes.POST("/:k_id/chat-enabled", knowledgebase.EnabledChat)
}
