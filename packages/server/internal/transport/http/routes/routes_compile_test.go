package routes

import (
	"net/http"
	"net/http/httptest"
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
	var _ func(*gin.RouterGroup, handler.MCPServerHandler, gin.HandlerFunc) = RegisterMCPServerRoutes
	var _ func(*gin.RouterGroup) = RegisterDebugRoutes
}

func TestRegisterMCPServerRoutesRegistersStandardPatchPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")

	RegisterMCPServerRoutes(api, handler.MCPServerHandler{}, func(c *gin.Context) {})

	for _, route := range router.Routes() {
		if route.Method == "PATCH" && route.Path == "/api/mcp/:mcp_id" {
			return
		}
	}
	t.Fatalf("PATCH /api/mcp/:mcp_id route was not registered; routes=%+v", router.Routes())
}

func TestRegisterMCPServerRoutesRegistersTogglePathOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")

	RegisterMCPServerRoutes(api, handler.MCPServerHandler{}, func(c *gin.Context) {})

	seen := map[string]bool{}
	for _, route := range router.Routes() {
		if route.Method == "POST" {
			seen[route.Path] = true
		}
	}
	if !seen["/api/mcp/:mcp_id/toggle"] {
		t.Fatalf("POST /api/mcp/:mcp_id/toggle route was not registered; routes=%+v", router.Routes())
	}
	if seen["/api/mcp/:mcp_id/troggle"] {
		t.Fatalf("POST /api/mcp/:mcp_id/troggle route should not be registered; routes=%+v", router.Routes())
	}
}

// TestRegisterKnowledgeBaseRoutesRegistersContractPaths 验证知识库列表与创建接口注册了 OpenAPI 声明的无尾斜杠路径。
func TestRegisterKnowledgeBaseRoutesRegistersContractPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")

	RegisterKnowledgeBaseRoutes(api, handler.KnowledgeBaseHandler{}, func(c *gin.Context) {})

	seen := map[string]bool{}
	for _, route := range router.Routes() {
		if route.Path == "/api/knowledge-base" {
			seen[route.Method] = true
		}
	}
	if !seen["GET"] || !seen["POST"] {
		t.Fatalf("contract routes GET/POST /api/knowledge-base were not both registered; routes=%+v", router.Routes())
	}

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		request := httptest.NewRequest(method, "/api/knowledge-base", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code == http.StatusTemporaryRedirect || response.Code == http.StatusPermanentRedirect {
			t.Fatalf("%s /api/knowledge-base redirected with status %d", method, response.Code)
		}
	}
}

// TestRegisterDocumentRoutesRegistersContractPath 验证文档列表接口注册了 OpenAPI 声明的无尾斜杠路径。
func TestRegisterDocumentRoutesRegistersContractPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")

	RegisterDocumentRoutes(api, handler.DocumentHandler{}, func(c *gin.Context) {})

	for _, route := range router.Routes() {
		if route.Method == "GET" && route.Path == "/api/document" {
			request := httptest.NewRequest(http.MethodGet, "/api/document?page=1&page_size=20", nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code == http.StatusTemporaryRedirect || response.Code == http.StatusPermanentRedirect {
				t.Fatalf("GET /api/document redirected with status %d", response.Code)
			}
			return
		}
	}
	t.Fatalf("contract route GET /api/document was not registered; routes=%+v", router.Routes())
}
