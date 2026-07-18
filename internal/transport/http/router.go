package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boxify/api-go/internal/config"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/boxify/api-go/internal/transport/http/middleware"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/transport/http/routes"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Dependencies struct {
	Svc                   *svc.ServiceContext
	EnableDebugPanicRoute bool
}

func NewRouter(deps Dependencies) *gin.Engine {
	gin.SetMode(gin.TestMode)
	response.RegisterValidatorTagNames()

	r := gin.New()
	r.Use(xlog.RecoveryMiddleware())
	r.Use(xlog.GinMiddleware())
	r.Use(cors())
	r.NoRoute(func(c *gin.Context) {
		response.FromError(c, xerr.NotFound("route not found"))
	})
	registerDocsRoutes(r, deps.Svc.Config.Docs)

	health := handler.NewHealthHandler(deps.Svc)
	auth := handler.NewAuthHandler(deps.Svc)
	chat := handler.NewChatHandler(deps.Svc)
	modelConfig := handler.NewModelConfigHandler(deps.Svc)
	conversation := handler.NewConversationHandler(deps.Svc)
	agentConfig := handler.NewAgentConfigHandler(deps.Svc)
	agentPersona := handler.NewAgentPersonaHandler(deps.Svc)
	mcpServer := handler.NewMCPServerHandler(deps.Svc)
	knowledgeBase := handler.NewKnowledgeBaseHandler(deps.Svc)
	document := handler.NewDocumentHandler(deps.Svc)
	image := handler.NewImageHandler(deps.Svc)
	tag := handler.NewTagHandler(deps.Svc)
	skill := handler.NewSkillHandler(deps.Svc)
	toolConfig := handler.NewToolConfigHandler(deps.Svc)
	gateway := handler.NewGatewayHandler(deps.Svc)

	authMiddleware := middleware.Auth(deps.Svc.TokenIssuer)

	api := r.Group("/api")
	routes.RegisterHealthRoutes(api, health)
	routes.RegisterAuthRoutes(api, auth, authMiddleware)
	routes.RegisterChatRoutes(api, chat, authMiddleware)
	routes.RegisterModelConfigRoutes(api, modelConfig, authMiddleware)
	routes.RegisterConversationRoutes(api, conversation, authMiddleware)
	routes.RegisterAgentConfigRoutes(api, agentConfig, authMiddleware)
	routes.RegisterAgentPersonaRoutes(api, agentPersona, authMiddleware)
	routes.RegisterMCPServerRoutes(api, mcpServer, authMiddleware)
	routes.RegisterKnowledgeBaseRoutes(api, knowledgeBase, authMiddleware)
	routes.RegisterDocumentRoutes(api, document, authMiddleware)
	routes.RegisterImageRoutes(api, image, authMiddleware)
	routes.RegisterTagRoutes(api, tag, authMiddleware)
	routes.RegisterSkillRoutes(api, skill, authMiddleware)
	routes.RegisterToolConfigRoutes(api, toolConfig, authMiddleware)
	routes.RegisterGatewayRoutes(api, gateway, authMiddleware)
	if deps.EnableDebugPanicRoute {
		routes.RegisterDebugRoutes(api)
	}
	return r
}

func registerDocsRoutes(r *gin.Engine, cfg config.DocsConfig) {
	if !cfg.Enabled {
		return
	}
	if cfg.Path == "" {
		cfg.Path = "/api/docs"
	}
	if cfg.SpecPath == "" {
		cfg.SpecPath = strings.TrimRight(cfg.Path, "/") + "/openapi.json"
	}
	docsPath := strings.TrimRight(cfg.Path, "/")
	ui := ginSwagger.WrapHandler(swaggerfiles.Handler, ginSwagger.URL(cfg.SpecPath))
	r.GET(docsPath, func(c *gin.Context) {
		c.Redirect(http.StatusFound, docsPath+"/index.html")
	})
	r.GET(docsPath+"/*any", func(c *gin.Context) {
		if strings.TrimPrefix(c.Param("any"), "/") == "openapi.json" {
			writeOpenAPISpec(c)
			return
		}
		ui(c)
	})
}

func writeOpenAPISpec(c *gin.Context) {
	data, err := os.ReadFile(filepath.Join("docs", "openapi.json"))
	if err != nil {
		data = []byte(`{"openapi":"3.0.3","info":{"title":"Cove API","version":"0.1.0"},"paths":{}}`)
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Max-Age", (12 * time.Hour).String())
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
