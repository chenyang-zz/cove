package routes

import (
	"testing"

	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func TestRouteRegistrationHelpersAreDefined(t *testing.T) {
	t.Helper()

	var _ func(*gin.RouterGroup, handler.HealthHandler) = RegisterHealthRoutes
	var _ func(*gin.RouterGroup, handler.AuthHandler, gin.HandlerFunc) = RegisterAuthRoutes
	var _ func(*gin.RouterGroup, handler.ChatHandler, gin.HandlerFunc) = RegisterChatRoutes
	var _ func(*gin.RouterGroup, handler.ModelConfigHandler, gin.HandlerFunc) = RegisterModelConfigRoutes
	var _ func(*gin.RouterGroup) = RegisterDebugRoutes
}
