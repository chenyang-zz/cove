/**
 * @Time   : 2026/6/29 11:23
 * @Author : chenyangzhao542@gmail.com
 * @File   : agentpersona.go
 **/

package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterAgentPersonaRoutes(api *gin.RouterGroup, agentPersona handler.AgentPersonaHandler, authMiddleware gin.HandlerFunc) {
	agentPersonaRoutes := api.Group("/agent-persona", authMiddleware)
	// @auth(user_id)
	// @description 查询智能体角色列表
	// @input request.ListAgentPersonasRequest
	// @output response.ListResponse[*response.AgentPersonaResponse]
	agentPersonaRoutes.GET("", agentPersona.ListAgentPersonas)
	// @auth(user_id)
	// @description 创建智能体角色
	// @input request.CreateAgentPersonaRequest
	// @output response.AgentPersonaResponse
	agentPersonaRoutes.POST("/", agentPersona.CreateAgentPersona)
	agentPersonaRoutes.POST("/create", agentPersona.CreateAgentPersona)
	// @auth(user_id)
	// @description 更新智能体角色
	// @input request.UpdateAgentPersonaRequest
	// @output response.AgentPersonaResponse
	agentPersonaRoutes.PATCH("/:persona_id", agentPersona.UpdateAgentPersona)
	agentPersonaRoutes.POST("/:persona_id/update", agentPersona.UpdateAgentPersona)
	// @auth(user_id)
	// @description 激活智能体角色
	// @input request.UriAgentPersonaIDRequest
	// @output response.AgentPersonaResponse
	agentPersonaRoutes.POST("/:persona_id/activate", agentPersona.ActivateAgentPersona)

	// @auth(user_id)
	// @description 删除智能体角色
	// @input request.UriAgentPersonaIDRequest
	agentPersonaRoutes.DELETE("/:persona_id", agentPersona.DeleteAgentPersona)
	agentPersonaRoutes.POST("/:persona_id/delete", agentPersona.DeleteAgentPersona)
}
