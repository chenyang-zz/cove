package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

// RegisterGatewayRoutes 注册 JWT 用户隔离的网关控制面路由。
func RegisterGatewayRoutes(api *gin.RouterGroup, gateway handler.GatewayHandler, auth gin.HandlerFunc) {
	routes := api.Group("/gateway", auth)
	// @auth(user_id)
	// @description 查询网关支持的 Provider、配置字段和能力矩阵
	// @output []response.GatewayProviderResponse
	routes.GET("/providers", gateway.Providers)
	// @auth(user_id)
	// @description 查询当前用户的渠道账号，凭据仅返回掩码
	// @output []*response.ChannelAccountResponse
	routes.GET("/accounts", gateway.ListAccounts)
	// @auth(user_id)
	// @description 创建渠道账号并加密保存凭据
	// @input request.CreateChannelAccountRequest
	// @output response.ChannelAccountResponse
	routes.POST("/accounts", gateway.CreateAccount)
	// @auth(user_id)
	// @description 查询渠道账号详情
	// @input request.UriGatewayIDRequest
	// @output response.ChannelAccountResponse
	routes.GET("/accounts/:id", gateway.GetAccount)
	// @auth(user_id)
	// @description 局部更新渠道账号并通知网关热重载
	// @input request.UpdateChannelAccountDocRequest
	// @output response.ChannelAccountResponse
	routes.PATCH("/accounts/:id", gateway.UpdateAccount)
	// @auth(user_id)
	// @description 删除渠道账号及其网关数据
	// @input request.UriGatewayIDRequest
	// @output response.GatewayStatusResponse
	routes.DELETE("/accounts/:id", gateway.DeleteAccount)
	// @auth(user_id)
	// @description 验证渠道账号凭据和平台连通性
	// @input request.UriGatewayIDRequest
	// @output response.GatewayStatusResponse
	routes.POST("/accounts/:id/test", gateway.TestAccount)
	// @auth(user_id)
	// @description 查询渠道账号的私聊配对请求
	// @input request.UriGatewayIDRequest
	// @output []*response.ChannelPairingResponse
	routes.GET("/accounts/:id/pairings", gateway.ListPairings)
	// @auth(user_id)
	// @description 批准私聊配对请求
	// @input request.UriGatewayIDRequest
	// @output response.ChannelPairingResponse
	routes.POST("/pairings/:id/approve", gateway.ApprovePairing)
	// @auth(user_id)
	// @description 拒绝并阻止私聊配对请求
	// @input request.UriGatewayIDRequest
	// @output response.ChannelPairingResponse
	routes.POST("/pairings/:id/deny", gateway.DenyPairing)
	// @auth(user_id)
	// @description 查询当前用户的确定性渠道路由绑定
	// @input request.ListChannelBindingsRequest
	// @output []*response.ChannelBindingResponse
	routes.GET("/bindings", gateway.ListBindings)
	// @auth(user_id)
	// @description 创建私聊或白名单群聊绑定
	// @input request.CreateChannelBindingRequest
	// @output response.ChannelBindingResponse
	routes.POST("/bindings", gateway.CreateBinding)
	// @auth(user_id)
	// @description 查询渠道路由绑定详情
	// @input request.UriGatewayIDRequest
	// @output response.ChannelBindingResponse
	routes.GET("/bindings/:id", gateway.GetBinding)
	// @auth(user_id)
	// @description 更新绑定的 Agent、提及门控或工具策略
	// @input request.UpdateChannelBindingDocRequest
	// @output response.ChannelBindingResponse
	routes.PATCH("/bindings/:id", gateway.UpdateBinding)
	// @auth(user_id)
	// @description 删除渠道路由绑定
	// @input request.UriGatewayIDRequest
	// @output response.GatewayStatusResponse
	routes.DELETE("/bindings/:id", gateway.DeleteBinding)
}
