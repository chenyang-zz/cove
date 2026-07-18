package routes

import (
	"testing"

	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

// TestRegisterGatewayRoutes 验证控制面完整注册账号、配对和绑定路由。
func TestRegisterGatewayRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterGatewayRoutes(router.Group("/api"), handler.GatewayHandler{}, func(c *gin.Context) {})
	want := map[string]bool{
		"GET /api/gateway/providers": true, "GET /api/gateway/accounts": true,
		"POST /api/gateway/accounts": true, "GET /api/gateway/accounts/:id": true,
		"PATCH /api/gateway/accounts/:id": true, "DELETE /api/gateway/accounts/:id": true,
		"POST /api/gateway/accounts/:id/test": true, "GET /api/gateway/accounts/:id/pairings": true,
		"POST /api/gateway/pairings/:id/approve": true, "POST /api/gateway/pairings/:id/deny": true,
		"GET /api/gateway/bindings": true, "POST /api/gateway/bindings": true,
		"GET /api/gateway/bindings/:id": true, "PATCH /api/gateway/bindings/:id": true,
		"DELETE /api/gateway/bindings/:id": true,
	}
	for _, route := range router.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	if len(want) != 0 {
		t.Fatalf("missing gateway routes: %v", want)
	}
}
