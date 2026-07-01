package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDirective(t *testing.T) {
	directive, ok := parseDirective("// routegen: auth user_id sse input=request.CreateBookRequest output=response.BookResponse event=domain.AgentEvent")
	if !ok {
		t.Fatal("parseDirective ok = false, want true")
	}
	if !directive.Auth || !directive.UserID {
		t.Fatalf("directive auth/user_id = %v/%v, want true/true", directive.Auth, directive.UserID)
	}
	if directive.Input != "request.CreateBookRequest" {
		t.Fatalf("input = %q", directive.Input)
	}
	if directive.Output != "response.BookResponse" {
		t.Fatalf("output = %q", directive.Output)
	}
	if !directive.SSE || directive.Event != "domain.AgentEvent" {
		t.Fatalf("sse/event = %v/%q, want true/domain.AgentEvent", directive.SSE, directive.Event)
	}
}

func TestParseDirectiveWithoutOutputKeepsOutputEmpty(t *testing.T) {
	directive, ok := parseDirective("// routegen: auth user_id")
	if !ok {
		t.Fatal("parseDirective ok = false, want true")
	}
	if directive.Output != "" {
		t.Fatalf("output = %q, want empty", directive.Output)
	}
}

func TestParseDirectiveGroupSupportsAtRoutegenProtocol(t *testing.T) {
	group := commentGroup(
		"// 创建图书。",
		"// 保存图书的基础信息。",
		"// @routegen",
		"// @auth",
		"// @userID",
		"// @input *request.CreateBookRequest",
		"// @output response.ListResponse[*response.BookResponse]",
	)

	directive, comments, ok := parseDirectiveGroup(group)
	if !ok {
		t.Fatal("parseDirectiveGroup ok = false, want true")
	}
	if !directive.Auth || !directive.UserID {
		t.Fatalf("directive auth/userID = %v/%v, want true/true", directive.Auth, directive.UserID)
	}
	if directive.Input != "*request.CreateBookRequest" {
		t.Fatalf("input = %q", directive.Input)
	}
	if directive.Output != "response.ListResponse[*response.BookResponse]" {
		t.Fatalf("output = %q", directive.Output)
	}
	wantComments := []string{"创建图书。", "保存图书的基础信息。"}
	if strings.Join(comments, "\n") != strings.Join(wantComments, "\n") {
		t.Fatalf("comments = %#v, want %#v", comments, wantComments)
	}
}

func TestParseDirectiveGroupSupportsNewAnnotationProtocol(t *testing.T) {
	group := commentGroup(
		"// 普通注释会被 description 覆盖。",
		"// @auth(user_id)",
		"// @description 重命名会话",
		"// 修改当前用户拥有的会话标题。",
		"// 标题不能为空。",
		"// @summary 重命名会话",
		"// @input request.RenameConversationRequest",
		"// @response ConversationResponse",
	)

	directive, comments, ok := parseDirectiveGroup(group)
	if !ok {
		t.Fatal("parseDirectiveGroup ok = false, want true")
	}
	if !directive.Auth || !directive.UserID {
		t.Fatalf("directive auth/userID = %v/%v, want true/true", directive.Auth, directive.UserID)
	}
	if directive.Input != "request.RenameConversationRequest" {
		t.Fatalf("input = %q", directive.Input)
	}
	if directive.Output != "response.ConversationResponse" {
		t.Fatalf("output = %q, want response.ConversationResponse", directive.Output)
	}
	if directive.Summary != "重命名会话" {
		t.Fatalf("summary = %q", directive.Summary)
	}
	wantDescription := []string{"重命名会话", "修改当前用户拥有的会话标题。", "标题不能为空。"}
	if strings.Join(comments, "\n") != strings.Join(wantDescription, "\n") {
		t.Fatalf("comments = %#v, want %#v", comments, wantDescription)
	}
}

func TestParseDirectiveGroupSupportsEventAnnotation(t *testing.T) {
	group := commentGroup(
		"// @auth(user_id)",
		"// @sse",
		"// @event domain.AgentEvent",
		"// @description 流式聊天",
		"// @input request.ChatStreamRequest",
	)

	directive, comments, ok := parseDirectiveGroup(group)
	if !ok {
		t.Fatal("parseDirectiveGroup ok = false, want true")
	}
	if !directive.SSE || directive.Event != "domain.AgentEvent" {
		t.Fatalf("sse/event = %v/%q, want true/domain.AgentEvent", directive.SSE, directive.Event)
	}
	if strings.Join(comments, "\n") != "流式聊天" {
		t.Fatalf("comments = %#v, want description", comments)
	}
}

func TestParseDirectiveGroupSupportsAuthWithoutUserIDAndResponseNormalization(t *testing.T) {
	group := commentGroup(
		"// @auth",
		"// @input request.ListConversationRequest",
		"// @response ListResponse[*response.ConversationResponse]",
	)

	directive, _, ok := parseDirectiveGroup(group)
	if !ok {
		t.Fatal("parseDirectiveGroup ok = false, want true")
	}
	if !directive.Auth || directive.UserID {
		t.Fatalf("directive auth/userID = %v/%v, want true/false", directive.Auth, directive.UserID)
	}
	if directive.Output != "response.ListResponse[*response.ConversationResponse]" {
		t.Fatalf("output = %q", directive.Output)
	}
}

func TestSnakeCaseKeepsAcronymsTogether(t *testing.T) {
	tests := map[string]string{
		"AgentConfig":           "agent_config",
		"CreateConversation":    "create_conversation",
		"GetMCPServerList":      "get_mcp_server_list",
		"HTTPRequest":           "http_request",
		"MCPServer":             "mcp_server",
		"MCPServerHandler":      "mcp_server_handler",
		"ModelConfig":           "model_config",
		"UriMCPServerIDRequest": "uri_mcp_server_id_request",
	}
	for input, want := range tests {
		if got := snakeCase(input); got != want {
			t.Fatalf("snakeCase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRenderTemplateMissingTemplateReturnsError(t *testing.T) {
	// 验证缺失模板时会返回包含模板名的明确错误。
	_, err := renderTemplate("missing.gotmpl", nil)
	if err == nil {
		t.Fatal("renderTemplate error = nil, want missing template error")
	}
	if !strings.Contains(err.Error(), "render template missing.gotmpl") {
		t.Fatalf("error = %v, want template name", err)
	}
}

func TestRenderModelBeforeCreateTemplate(t *testing.T) {
	// 验证 BeforeCreate 模板会生成 UUID 初始化 hook。
	out, err := renderTemplate("model_before_create.gotmpl", map[string]any{
		"Receiver": "m",
		"Model":    "Message",
	})
	if err != nil {
		t.Fatalf("renderTemplate error = %v", err)
	}
	for _, want := range []string{
		"func (m *Message) BeforeCreate(tx *gorm.DB) error",
		"ensureUUID(&m.ID)",
		"return nil",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("template output missing %q:\n%s", want, out)
		}
	}
}

func TestScanRoutesParsesRouteDirectiveAndHandlerType(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.CreateBookRequest output=response.BookResponse
	bookRoutes.POST("", book.Create)
	// routegen: auth user_id output=[]*response.BookResponse
	bookRoutes.GET("", book.List)
}
`)

	routes, err := scanRoutes(root)
	if err != nil {
		t.Fatalf("scanRoutes error = %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2", len(routes))
	}
	create := routes[0]
	if create.HTTPMethod != "POST" || create.HandlerType != "BookHandler" || create.HandlerMethod != "Create" {
		t.Fatalf("create route = %+v", create)
	}
	if create.Domain != "book" || create.Directive.Input != "request.CreateBookRequest" || create.Directive.Output != "response.BookResponse" {
		t.Fatalf("create route directive/domain = %+v", create)
	}
	list := routes[1]
	if list.HTTPMethod != "GET" || list.HandlerMethod != "List" || list.Directive.Output != "[]*response.BookResponse" {
		t.Fatalf("list route = %+v", list)
	}
}

func TestScanRoutesFindsHandlerAfterMiddlewareArgs(t *testing.T) {
	// 验证扫描路由时能跳过 middleware 参数，并保留 Gin group 前缀形成完整路径。
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books")
	// routegen: auth user_id input=request.UpdateBookRequest output=response.BookResponse
	bookRoutes.PUT("/:id", authMiddleware, book.Update)
}
`)

	routes, err := scanRoutes(root)
	if err != nil {
		t.Fatalf("scanRoutes error = %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}
	if routes[0].HandlerType != "BookHandler" || routes[0].HandlerMethod != "Update" {
		t.Fatalf("route = %+v, want BookHandler.Update", routes[0])
	}
	if routes[0].Path != "/books/:id" {
		t.Fatalf("route path = %q, want /books/:id", routes[0].Path)
	}
}

func TestScanRoutesParsesCustomLogicComment(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	bookRoutes := api.Group("/books")
	// 创建图书。
	// 保存图书的基础信息。
	// routegen: input=request.CreateBookRequest output=response.BookResponse
	bookRoutes.POST("", book.Create)
}
`)

	routes, err := scanRoutes(root)
	if err != nil {
		t.Fatalf("scanRoutes error = %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}
	if got := strings.Join(routes[0].CommentLines, "\n"); got != "创建图书。\n保存图书的基础信息。" {
		t.Fatalf("comment lines = %q", got)
	}
}

func TestScanRoutesParsesAtRoutegenProtocolAndMultilineComment(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	bookRoutes := api.Group("/books")
	// 创建图书。
	// 保存图书的基础信息。
	// @routegen
	// @auth
	// @userID
	// @input request.CreateBookRequest
	// @output response.BookResponse
	bookRoutes.POST("", book.Create)
}
`)

	routes, err := scanRoutes(root)
	if err != nil {
		t.Fatalf("scanRoutes error = %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}
	route := routes[0]
	if !route.Directive.Auth || !route.Directive.UserID {
		t.Fatalf("directive auth/userID = %v/%v, want true/true", route.Directive.Auth, route.Directive.UserID)
	}
	if route.Directive.Input != "request.CreateBookRequest" || route.Directive.Output != "response.BookResponse" {
		t.Fatalf("directive = %+v", route.Directive)
	}
	if got := strings.Join(route.CommentLines, "\n"); got != "创建图书。\n保存图书的基础信息。" {
		t.Fatalf("comment lines = %q", got)
	}
}

func TestGenerateCreatesMissingHandlerAndLogic(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.CreateBookRequest output=response.BookResponse
	bookRoutes.POST("", book.Create)
}
`)

	report, err := GenerateRoutes(root)
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"type BookHandler struct",
		"func NewBookHandler(svcCtx *svc.ServiceContext) BookHandler",
		"func (h BookHandler) Create(c *gin.Context)",
		"var body request.CreateBookRequest",
		"c.ShouldBindJSON(&body)",
		"userID, err := util.UserIDFromContext(c.Request.Context())",
		"booklogic.NewCreateLogic(c.Request.Context(), h.svc).Create(userID, &body)",
		"response.OK(c, out)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}

	logicFile := readFile(t, root, "internal/logic/book/create.go")
	if strings.Contains(logicFile, generatedHeader) {
		t.Fatalf("logic file contains generated header:\n%s", logicFile)
	}
	for _, want := range []string{
		"// CreateLogic contains the create use case.",
		"type CreateLogic struct",
		"// NewCreateLogic creates a CreateLogic.",
		`xlog.Component("logic.book.create")`,
		"// Create handles the create use case.",
		"func (l *CreateLogic) Create(userID uuid.UUID, input *request.CreateBookRequest) (*response.BookResponse, error)",
		"return nil, nil",
	} {
		if !strings.Contains(logicFile, want) {
			t.Fatalf("logic file missing %q:\n%s", want, logicFile)
		}
	}
	assertReportContains(t, report, FileAdded, "internal/transport/http/handler/book.go")
	assertReportContains(t, report, FileAdded, "internal/logic/book/create.go")
}

func TestGenerateFailsForSSEWithoutEvent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/chat.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterChatRoutes(api *gin.RouterGroup, chat handler.ChatHandler, authMiddleware gin.HandlerFunc) {
	chatRoutes := api.Group("/chat", authMiddleware)
	// @auth(user_id)
	// @sse
	// @input request.ChatStreamRequest
	chatRoutes.POST("/stream", chat.ChatStream)
}
`)

	_, err := GenerateRoutes(root)
	if err == nil {
		t.Fatal("GenerateRoutes error = nil, want missing @event error")
	}
	if !strings.Contains(err.Error(), "uses @sse but missing @event <GoType>") {
		t.Fatalf("error = %v, want missing @event hint", err)
	}
}

func TestGenerateCreatesSSEHandlerAndLogicWithEvent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/chat.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterChatRoutes(api *gin.RouterGroup, chat handler.ChatHandler, authMiddleware gin.HandlerFunc) {
	chatRoutes := api.Group("/chat", authMiddleware)
	// @auth(user_id)
	// @sse
	// @event domain.AgentEvent
	// @description 流式聊天
	// @input request.ChatStreamRequest
	chatRoutes.POST("/stream", chat.ChatStream)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("GenerateRoutes error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/chat.go")
	for _, want := range []string{
		"events, err := chatlogic.NewChatStreamLogic(c.Request.Context(), h.svc).ChatStream(userID, &body)",
		"response.StreamEvents(c, events)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	for _, forbidden := range []string{"_ = events", "response.OK(c, map[string]any{})"} {
		if strings.Contains(handlerFile, forbidden) {
			t.Fatalf("handler file should not contain %q:\n%s", forbidden, handlerFile)
		}
	}

	logicFile := readFile(t, root, "internal/logic/chat/chat_stream.go")
	for _, want := range []string{
		`"github.com/boxify/api-go/internal/domain"`,
		"func (l *ChatStreamLogic) ChatStream(userID uuid.UUID, input *request.ChatStreamRequest) (<-chan domain.AgentEvent, error)",
		"return nil, nil",
	} {
		if !strings.Contains(logicFile, want) {
			t.Fatalf("logic file missing %q:\n%s", want, logicFile)
		}
	}
}

func TestGenerateCreatesSSELogicWithResponseEvent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/chat.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterChatRoutes(api *gin.RouterGroup, chat handler.ChatHandler) {
	chatRoutes := api.Group("/chat")
	// @sse
	// @event response.ChatEvent
	chatRoutes.GET("/events", chat.Events)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("GenerateRoutes error = %v", err)
	}

	logicFile := readFile(t, root, "internal/logic/chat/events.go")
	for _, want := range []string{
		`"github.com/boxify/api-go/internal/transport/http/response"`,
		"func (l *EventsLogic) Events() (<-chan response.ChatEvent, error)",
	} {
		if !strings.Contains(logicFile, want) {
			t.Fatalf("logic file missing %q:\n%s", want, logicFile)
		}
	}
}

func TestGenerateRouteUsesAcronymAwareLogicFilename(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/mcp.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterMCPServerRoutes(api *gin.RouterGroup, mcp handler.MCPServerHandler, authMiddleware gin.HandlerFunc) {
	mcpRoutes := api.Group("/mcp", authMiddleware)
	// @auth(user_id)
	// @response ListResponse[*response.MCPServerResponse]
	mcpRoutes.GET("/", mcp.GetMCPServerList)
}
`)

	report, err := GenerateRoutes(root)
	if err != nil {
		t.Fatalf("GenerateRoutes error = %v", err)
	}

	assertReportContains(t, report, FileAdded, "internal/logic/mcpserver/get_mcp_server_list.go")
	logicFile := readFile(t, root, "internal/logic/mcpserver/get_mcp_server_list.go")
	if !strings.Contains(logicFile, "func (l *GetMCPServerListLogic) GetMCPServerList(userID uuid.UUID) (*response.ListResponse[*response.MCPServerResponse], error)") {
		t.Fatalf("logic file has unexpected signature:\n%s", logicFile)
	}
	if _, err := os.Stat(filepath.Join(root, "internal/logic/mcpserver/get_m_c_p_server_list.go")); err == nil {
		t.Fatal("generated old acronym-split logic filename")
	}
}

func TestGenerateUsesCustomLogicMethodComment(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	bookRoutes := api.Group("/books")
	// 创建图书。
	// 保存图书的基础信息。
	// routegen: input=request.CreateBookRequest output=response.BookResponse
	bookRoutes.POST("", book.Create)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	logicFile := readFile(t, root, "internal/logic/book/create.go")
	for _, want := range []string{
		"// CreateLogic contains the create use case.",
		"// NewCreateLogic creates a CreateLogic.",
		"// Create 创建图书。\n// 保存图书的基础信息。",
	} {
		if !strings.Contains(logicFile, want) {
			t.Fatalf("logic file missing %q:\n%s", want, logicFile)
		}
	}
}

func TestGenerateUsesAtRoutegenMultilineLogicMethodComment(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	bookRoutes := api.Group("/books")
	// 创建图书。
	// 保存图书的基础信息。
	// @routegen
	// @input request.CreateBookRequest
	// @output response.BookResponse
	bookRoutes.POST("", book.Create)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	logicFile := readFile(t, root, "internal/logic/book/create.go")
	if !strings.Contains(logicFile, "// Create 创建图书。\n// 保存图书的基础信息。") {
		t.Fatalf("logic file missing multiline custom comment:\n%s", logicFile)
	}
}

func TestGenerateUsesMultilineDescriptionMethodComment(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/conversation.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterConversationRoutes(api *gin.RouterGroup, conversation handler.ConversationHandler, authMiddleware gin.HandlerFunc) {
	conversationRoutes := api.Group("/conversation", authMiddleware)
	// 普通注释不会进入 GoDoc。
	// @auth(user_id)
	// @description 重命名会话
	// 修改当前用户拥有的会话标题。
	// 标题不能为空。
	// @summary 会话标题更新
	// @input request.RenameConversationRequest
	// @response ConversationResponse
	conversationRoutes.PATCH("/:conversation_id", conversation.RenameConversation)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	logicFile := readFile(t, root, "internal/logic/conversation/rename_conversation.go")
	for _, want := range []string{
		"// RenameConversation 重命名会话\n// 修改当前用户拥有的会话标题。\n// 标题不能为空。",
		"func (l *RenameConversationLogic) RenameConversation(userID uuid.UUID, input *request.RenameConversationRequest) (*response.ConversationResponse, error)",
	} {
		if !strings.Contains(logicFile, want) {
			t.Fatalf("logic file missing %q:\n%s", want, logicFile)
		}
	}
	if strings.Contains(logicFile, "普通注释不会进入 GoDoc") {
		t.Fatalf("description should override normal comments:\n%s", logicFile)
	}
	if strings.Contains(logicFile, "会话标题更新") {
		t.Fatalf("summary should not be used as logic GoDoc:\n%s", logicFile)
	}
}

func TestGenerateDoesNotDuplicateMethodNameInDescription(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/conversation.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterConversationRoutes(api *gin.RouterGroup, conversation handler.ConversationHandler) {
	conversationRoutes := api.Group("/conversation")
	// @description RenameConversation 重命名会话
	// @input request.RenameConversationRequest
	// @response ConversationResponse
	conversationRoutes.PATCH("/:conversation_id", conversation.RenameConversation)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	logicFile := readFile(t, root, "internal/logic/conversation/rename_conversation.go")
	if !strings.Contains(logicFile, "// RenameConversation 重命名会话") {
		t.Fatalf("logic file missing description comment:\n%s", logicFile)
	}
	if strings.Contains(logicFile, "// RenameConversation RenameConversation") {
		t.Fatalf("logic file duplicated method name:\n%s", logicFile)
	}
}

func TestGenerateDoesNotDuplicateMethodNameInCustomLogicComment(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	bookRoutes := api.Group("/books")
	// Create 创建图书。
	// routegen: input=request.CreateBookRequest output=response.BookResponse
	bookRoutes.POST("", book.Create)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	logicFile := readFile(t, root, "internal/logic/book/create.go")
	if !strings.Contains(logicFile, "// Create 创建图书。") {
		t.Fatalf("logic file missing custom comment:\n%s", logicFile)
	}
	if strings.Contains(logicFile, "// Create Create") {
		t.Fatalf("logic file duplicated method name:\n%s", logicFile)
	}
}

func TestGenerateUsesQueryBindingForGETInput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.ListBooksRequest output=response.ListBooksResponse
	bookRoutes.GET("", book.List)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"var query request.ListBooksRequest",
		"c.ShouldBindQuery(&query)",
		"booklogic.NewListLogic(c.Request.Context(), h.svc).List(userID, &query)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Contains(handlerFile, "ShouldBindJSON(&query)") || strings.Contains(handlerFile, "var body request.ListBooksRequest") {
		t.Fatalf("GET input should use query binding only:\n%s", handlerFile)
	}
}

func TestGenerateUsesURIAndQueryBindingForGETInputWithURIParam(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.ListBookChaptersRequest output=response.ListBookChaptersResponse
	bookRoutes.GET("/:id/chapters", book.ListChapters)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"var query request.ListBookChaptersRequest",
		"c.ShouldBindUri(&query)",
		"c.ShouldBindQuery(&query)",
		"booklogic.NewListChaptersLogic(c.Request.Context(), h.svc).ListChapters(userID, &query)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Index(handlerFile, "ShouldBindUri(&query)") > strings.Index(handlerFile, "ShouldBindQuery(&query)") {
		t.Fatalf("URI binding should be generated before query binding:\n%s", handlerFile)
	}
}

func TestGenerateUsesURIAndJSONBindingForNonGETInputWithURIParam(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

type UriOnlyRequest struct {
	ID string `+"`uri:\"id\" binding:\"required\"`"+`
}

type UpdateBookRequest struct {
	UriOnlyRequest
	Title string `+"`json:\"title\" binding:\"required\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.UpdateBookRequest output=response.BookResponse
	bookRoutes.PUT("/:id", book.Update)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"var body request.UpdateBookRequest",
		"c.ShouldBindUri(&body.UriOnlyRequest)",
		"c.ShouldBindJSON(&body)",
		"booklogic.NewUpdateLogic(c.Request.Context(), h.svc).Update(userID, &body)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Contains(handlerFile, "ShouldBindUri(&body)") {
		t.Fatalf("composite DTO should bind URI-only embedded struct:\n%s", handlerFile)
	}
	if strings.Index(handlerFile, "ShouldBindUri(&body.UriOnlyRequest)") > strings.Index(handlerFile, "ShouldBindJSON(&body)") {
		t.Fatalf("URI binding should be generated before JSON binding:\n%s", handlerFile)
	}
}

func TestGenerateUsesEmbeddedURIOnlyBindingForRequiredJSONBody(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/request/conversation.go", `package request

type UriConversationIDRequest struct {
	ConversationID string `+"`uri:\"conversation_id\" binding:\"required\"`"+`
}

type RenameConversationRequest struct {
	UriConversationIDRequest
	Title string `+"`json:\"title\" binding:\"required,min=1,max=256\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/routes/conversation.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterConversationRoutes(api *gin.RouterGroup, conversation handler.ConversationHandler, authMiddleware gin.HandlerFunc) {
	conversationRoutes := api.Group("/conversation", authMiddleware)
	// routegen: auth user_id input=request.RenameConversationRequest output=response.ConversationResponse
	conversationRoutes.PATCH("/:conversation_id", conversation.RenameConversation)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/conversation.go")
	for _, want := range []string{
		"var body request.RenameConversationRequest",
		"c.ShouldBindUri(&body.UriConversationIDRequest)",
		"c.ShouldBindJSON(&body)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Contains(handlerFile, "ShouldBindUri(&body)") {
		t.Fatalf("composite DTO should not bind complete body for URI:\n%s", handlerFile)
	}
}

func TestGenerateSkipsJSONBindingForURIOnlyDELETEInput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

type UriOnlyRequest struct {
	ID string `+"`uri:\"id\" binding:\"required\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.UriOnlyRequest
	bookRoutes.DELETE("/:id", book.Delete)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"var body request.UriOnlyRequest",
		"c.ShouldBindUri(&body)",
		"booklogic.NewDeleteLogic(c.Request.Context(), h.svc).Delete(userID, &body)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Contains(handlerFile, "ShouldBindJSON") {
		t.Fatalf("URI-only input should not bind JSON:\n%s", handlerFile)
	}
}

func TestGenerateSkipsJSONBindingForURIOnlyPOSTInput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

type UriOnlyRequest struct {
	ID string `+"`uri:\"id\" binding:\"required\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.UriOnlyRequest output=response.BookResponse
	bookRoutes.POST("/:id/default", book.SetDefault)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"var body request.UriOnlyRequest",
		"c.ShouldBindUri(&body)",
		"booklogic.NewSetDefaultLogic(c.Request.Context(), h.svc).SetDefault(userID, &body)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Contains(handlerFile, "ShouldBindJSON") {
		t.Fatalf("URI-only input should not bind JSON:\n%s", handlerFile)
	}
}

func TestGenerateUsesMultipartBindingForUploadInput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

import "mime/multipart"

type UploadBookRequest struct {
	File *multipart.FileHeader `+"`form:\"file\" binding:\"required\"`"+`
	Title string `+"`form:\"title\" binding:\"required\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.UploadBookRequest output=response.BookResponse
	bookRoutes.POST("/upload", book.Upload)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"var body request.UploadBookRequest",
		"c.ShouldBind(&body)",
		"booklogic.NewUploadLogic(c.Request.Context(), h.svc).Upload(userID, &body)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Contains(handlerFile, "ShouldBindJSON") {
		t.Fatalf("multipart input should not bind JSON:\n%s", handlerFile)
	}
}

func TestGenerateUsesURIAndMultipartBindingForUploadInput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

import "mime/multipart"

type UriOnlyRequest struct {
	ID string `+"`uri:\"id\" binding:\"required\"`"+`
}

type UploadBookCoverRequest struct {
	UriOnlyRequest
	Cover multipart.FileHeader `+"`form:\"cover\" binding:\"required\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.UploadBookCoverRequest output=response.BookResponse
	bookRoutes.POST("/:id/cover", book.UploadCover)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"var body request.UploadBookCoverRequest",
		"c.ShouldBindUri(&body.UriOnlyRequest)",
		"c.ShouldBind(&body)",
		"booklogic.NewUploadCoverLogic(c.Request.Context(), h.svc).UploadCover(userID, &body)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Contains(handlerFile, "ShouldBindUri(&body)") {
		t.Fatalf("composite multipart DTO should bind URI-only embedded struct:\n%s", handlerFile)
	}
	if strings.Index(handlerFile, "ShouldBindUri(&body.UriOnlyRequest)") > strings.Index(handlerFile, "ShouldBind(&body)") {
		t.Fatalf("URI binding should be generated before multipart binding:\n%s", handlerFile)
	}
	if strings.Contains(handlerFile, "ShouldBindJSON") {
		t.Fatalf("multipart input should not bind JSON:\n%s", handlerFile)
	}
}

func TestGenerateUsesEmbeddedURIOnlyBindingForGETQueryInput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

type UriProjectIDRequest struct {
	ProjectID string `+"`uri:\"project_id\" binding:\"required\"`"+`
}

type ListProjectBooksRequest struct {
	UriProjectIDRequest
	Type string `+"`form:\"type\" binding:\"omitempty\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/projects", authMiddleware)
	// routegen: auth user_id input=request.ListProjectBooksRequest output=response.ListBooksResponse
	bookRoutes.GET("/:project_id/books", book.ListProjectBooks)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"var query request.ListProjectBooksRequest",
		"c.ShouldBindUri(&query.UriProjectIDRequest)",
		"c.ShouldBindQuery(&query)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Contains(handlerFile, "ShouldBindUri(&query)") {
		t.Fatalf("composite query DTO should bind URI-only embedded struct:\n%s", handlerFile)
	}
	if strings.Index(handlerFile, "ShouldBindUri(&query.UriProjectIDRequest)") > strings.Index(handlerFile, "ShouldBindQuery(&query)") {
		t.Fatalf("URI binding should be generated before query binding:\n%s", handlerFile)
	}
}

func TestGenerateKeepsCompleteDTOURISelectorForDirectURIFields(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

type RenameBookRequest struct {
	ID string `+"`uri:\"id\" binding:\"required\"`"+`
	Title string `+"`json:\"title\" binding:\"required\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	bookRoutes := api.Group("/books")
	// routegen: input=request.RenameBookRequest output=response.BookResponse
	bookRoutes.PATCH("/:id", book.Rename)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	if !strings.Contains(handlerFile, "c.ShouldBindUri(&body)") {
		t.Fatalf("direct URI field DTO should bind complete body:\n%s", handlerFile)
	}
}

func TestGenerateFailsForMultipleEmbeddedURIOnlyRequests(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

type UriBookIDRequest struct {
	BookID string `+"`uri:\"book_id\" binding:\"required\"`"+`
}

type UriChapterIDRequest struct {
	ChapterID string `+"`uri:\"chapter_id\" binding:\"required\"`"+`
}

type UpdateChapterRequest struct {
	UriBookIDRequest
	UriChapterIDRequest
	Title string `+"`json:\"title\" binding:\"required\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	bookRoutes := api.Group("/books")
	// routegen: input=request.UpdateChapterRequest output=response.BookResponse
	bookRoutes.PATCH("/:book_id/chapters/:chapter_id", book.UpdateChapter)
}
`)

	_, err := GenerateRoutes(root)
	if err == nil {
		t.Fatal("Generate error = nil, want multiple embedded URI-only error")
	}
	if !strings.Contains(err.Error(), "multiple embedded URI-only") {
		t.Fatalf("Generate error = %v, want multiple embedded URI-only message", err)
	}
}

func TestGenerateUsesMultipartBindingForMultiFileUploadInput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

import "mime/multipart"

type UploadBookImagesRequest struct {
	Images []*multipart.FileHeader `+"`form:\"images\" binding:\"required\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.UploadBookImagesRequest output=response.BookResponse
	bookRoutes.POST("/images", book.UploadImages)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	if !strings.Contains(handlerFile, "c.ShouldBind(&body)") {
		t.Fatalf("multipart input should bind multipart form:\n%s", handlerFile)
	}
	if strings.Contains(handlerFile, "ShouldBindJSON") {
		t.Fatalf("multipart input should not bind JSON:\n%s", handlerFile)
	}
}

func TestGenerateKeepsJSONBindingWhenRequestDTOCannotBeParsed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id input=request.UnknownRequest output=response.BookResponse
	bookRoutes.POST("/:id", book.Update)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	if !strings.Contains(handlerFile, "c.ShouldBindJSON(&body)") {
		t.Fatalf("unknown input should keep JSON binding fallback:\n%s", handlerFile)
	}
}

func TestGenerateCreatesNoOutputLogicWithOnlyErrorReturn(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books", authMiddleware)
	// routegen: auth user_id
	bookRoutes.DELETE("/:id", book.Delete)
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"if err := booklogic.NewDeleteLogic(c.Request.Context(), h.svc).Delete(userID); err != nil",
		"response.OK(c, nil)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Contains(handlerFile, "out, err :=") {
		t.Fatalf("handler file should not declare output value:\n%s", handlerFile)
	}
	if strings.Contains(handlerFile, "ShouldBindUri") {
		t.Fatalf("handler without input should not bind uri:\n%s", handlerFile)
	}

	logicFile := readFile(t, root, "internal/logic/book/delete.go")
	for _, want := range []string{
		"func (l *DeleteLogic) Delete(userID uuid.UUID) error",
		"return nil",
	} {
		if !strings.Contains(logicFile, want) {
			t.Fatalf("logic file missing %q:\n%s", want, logicFile)
		}
	}
	if strings.Contains(logicFile, "transport/http/response") {
		t.Fatalf("logic file should not import response package:\n%s", logicFile)
	}
}

func TestGenerateSkipsExistingHandlerMethodAndLogicFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	// routegen: output=response.BookResponse
	api.GET("/books/:id", book.Show)
}
`)
	existingHandler := `package handler

import "github.com/gin-gonic/gin"

type BookHandler struct{}

func (h BookHandler) Show(c *gin.Context) {}
`
	writeFile(t, root, "internal/transport/http/handler/book.go", existingHandler)
	existingLogic := `package book

type ShowLogic struct{}
`
	writeFile(t, root, "internal/logic/book/show.go", existingLogic)

	report, err := GenerateRoutes(root)
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	if got := readFile(t, root, "internal/transport/http/handler/book.go"); got != existingHandler {
		t.Fatalf("existing handler changed:\n%s", got)
	}
	if got := readFile(t, root, "internal/logic/book/show.go"); got != existingLogic {
		t.Fatalf("existing logic changed:\n%s", got)
	}
	assertReportContains(t, report, FileSkipped, "internal/transport/http/handler/book.go")
	assertReportContains(t, report, FileSkipped, "internal/logic/book/show.go")
}

func TestGenerateAppendsMissingMethodToExistingHandlerFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	// routegen: output=response.BookResponse
	api.GET("/books/:id", book.Show)
}
`)
	writeFile(t, root, "internal/transport/http/handler/book.go", `package handler

import "github.com/boxify/api-go/internal/svc"

type BookHandler struct {
	svc *svc.ServiceContext
}
`)

	report, err := GenerateRoutes(root)
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	if !strings.Contains(handlerFile, "func (h BookHandler) Show(c *gin.Context)") {
		t.Fatalf("generated handler missing Show:\n%s", handlerFile)
	}
	assertReportContains(t, report, FileModified, "internal/transport/http/handler/book.go")
}

func TestGenerateMethodOnlyHandlerDoesNotImportSvc(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	// routegen: input=request.UpdateBookRequest output=response.BookResponse
	api.PUT("/books/:id", book.Update)
}
`)
	writeFile(t, root, "internal/transport/http/handler/book.go", `package handler

import "github.com/boxify/api-go/internal/svc"

type BookHandler struct {
	svc *svc.ServiceContext
}
`)

	if _, err := GenerateRoutes(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	if strings.Count(handlerFile, `internal/svc`) != 1 {
		t.Fatalf("method-only generated handler svc import count is not 1:\n%s", handlerFile)
	}
	if !strings.Contains(handlerFile, "func (h BookHandler) Update(c *gin.Context)") {
		t.Fatalf("generated handler missing Update:\n%s", handlerFile)
	}
}

func TestGenerateReportsUnchangedGeneratedFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	// routegen: output=response.BookResponse
	api.GET("/books/:id", book.Show)
}
`)

	report, err := GenerateRoutes(root)
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if !report.Has(FileAdded, "internal/transport/http/handler/book.go") {
		t.Fatalf("first report = %+v, want added handler", report)
	}

	report, err = GenerateRoutes(root)
	if err != nil {
		t.Fatalf("second Generate error = %v", err)
	}
	assertReportContains(t, report, FileSkipped, "internal/transport/http/handler/book.go")
}

func TestPrintReportUsesColorAndCanDisableColor(t *testing.T) {
	report := Report{Files: []FileChange{
		{Kind: FileAdded, Path: "internal/transport/http/handler/book.go"},
		{Kind: FileModified, Path: "internal/logic/book/create.go"},
		{Kind: FileSkipped, Path: "internal/logic/book/list.go"},
	}}

	var colored bytes.Buffer
	printReport(&colored, report, true)
	coloredOut := colored.String()
	for _, want := range []string{
		"\x1b[32m+ internal/transport/http/handler/book.go\x1b[0m",
		"\x1b[33m~ internal/logic/book/create.go\x1b[0m",
		"\x1b[90m= internal/logic/book/list.go\x1b[0m",
	} {
		if !strings.Contains(coloredOut, want) {
			t.Fatalf("colored output missing %q:\n%s", want, coloredOut)
		}
	}

	var plain bytes.Buffer
	printReport(&plain, report, false)
	plainOut := plain.String()
	if strings.Contains(plainOut, "\x1b[") {
		t.Fatalf("plain output contains ANSI code:\n%s", plainOut)
	}
	if !strings.Contains(plainOut, "+ internal/transport/http/handler/book.go") {
		t.Fatalf("plain output missing added file:\n%s", plainOut)
	}
}

func TestGenerateRoutesDryRunDoesNotWriteFiles(t *testing.T) {
	// 验证 dry-run 只报告将要新增的文件，不实际写入磁盘。
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	// routegen: output=response.BookResponse
	api.GET("/books/:id", book.Show)
}
`)

	report, err := GenerateRoutesWithOptions(RouteOptions{Root: root, DryRun: true})
	if err != nil {
		t.Fatalf("GenerateRoutesWithOptions error = %v", err)
	}
	if report.Mode != ModeDryRun {
		t.Fatalf("mode = %q, want %q", report.Mode, ModeDryRun)
	}
	assertReportContains(t, report, FileWouldAdd, "internal/transport/http/handler/book.go")
	assertReportContains(t, report, FileWouldAdd, "internal/logic/book/show.go")
	if _, err := os.Stat(filepath.Join(root, "internal/transport/http/handler/book.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("handler file stat err = %v, want not exist", err)
	}
}

func TestRunRouteCommandCheckFailsWhenChangesNeeded(t *testing.T) {
	// 验证 check 模式发现待生成文件时返回专用错误，方便 CI 失败。
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	// routegen: output=response.BookResponse
	api.GET("/books/:id", book.Show)
}
`)

	report, err := runRouteCommand([]string{"-root", root, "--check", "--no-color"})
	if !errors.Is(err, ErrCheckFailed) {
		t.Fatalf("runRouteCommand error = %v, want ErrCheckFailed", err)
	}
	assertReportContains(t, report, FileWouldAdd, "internal/transport/http/handler/book.go")
	if _, err := os.Stat(filepath.Join(root, "internal/transport/http/handler/book.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("handler file stat err = %v, want not exist", err)
	}
}

func TestGenerateDocsCreatesOpenAPISpec(t *testing.T) {
	// 验证 docs 生成器能从 route 指令和 DTO tag 生成 OpenAPI 路径、参数、请求体和鉴权信息。
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler, authMiddleware gin.HandlerFunc) {
	bookRoutes := api.Group("/books")
	// @auth(user_id)
	// @summary 创建图书
	// @description 创建图书。
	// 保存图书基础信息。
	// @tag 图书
	// @input request.CreateBookRequest
	// @response BookResponse
	bookRoutes.POST("", authMiddleware, book.Create)
	// @auth(user_id)
	// @summary 查询图书章节
	// @input request.ListBookChaptersRequest
	// @response ListResponse[*response.ChapterResponse]
	bookRoutes.GET("/:book_id/chapters", authMiddleware, book.ListChapters)
}
`)
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

type CreateBookRequest struct {
	Title string `+"`json:\"title\" binding:\"required,min=1,max=128\"`"+`
	Kind  string `+"`json:\"kind\" binding:\"omitempty,oneof=novel essay\"`"+`
}

type ListBookChaptersRequest struct {
	BookID string `+"`uri:\"book_id\" binding:\"required\"`"+`
	Page   int    `+"`form:\"page\" binding:\"omitempty,min=1\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/response/book.go", `package response

type BookResponse struct {
	ID    string `+"`json:\"id\"`"+`
	Title string `+"`json:\"title\"`"+`
}

type ChapterResponse struct {
	ID string `+"`json:\"id\"`"+`
}
`)

	report, err := GenerateDocs(DocsOptions{Root: root})
	if err != nil {
		t.Fatalf("GenerateDocs error = %v", err)
	}
	assertReportContains(t, report, FileAdded, "docs/openapi.json")

	var spec map[string]any
	if err := json.Unmarshal([]byte(readFile(t, root, "docs/openapi.json")), &spec); err != nil {
		t.Fatalf("openapi json invalid: %v", err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Fatalf("openapi = %v, want 3.0.3", spec["openapi"])
	}
	paths := spec["paths"].(map[string]any)
	create := paths["/api/books"].(map[string]any)["post"].(map[string]any)
	if create["summary"] != "创建图书" || create["description"] != "创建图书。\n保存图书基础信息。" {
		t.Fatalf("create operation = %+v", create)
	}
	if len(create["security"].([]any)) == 0 {
		t.Fatalf("security = %+v, want bearer auth", create["security"])
	}
	if create["requestBody"] == nil {
		t.Fatalf("requestBody missing: %+v", create)
	}
	list := paths["/api/books/{book_id}/chapters"].(map[string]any)["get"].(map[string]any)
	params := list["parameters"].([]any)
	if len(params) != 2 {
		t.Fatalf("parameters = %+v, want path and query params", params)
	}
}

func TestGenerateDocsHandlesMultipartAndSSE(t *testing.T) {
	// 验证 multipart 文件上传和 SSE 路由会生成对应的 content type。
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/document.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterDocumentRoutes(api *gin.RouterGroup, doc handler.DocumentHandler, chat handler.ChatHandler, authMiddleware gin.HandlerFunc) {
	// @auth(user_id)
	// @input request.UploadDocumentRequest
	// @response DocumentResponse
	api.POST("/documents", authMiddleware, doc.Upload)
	// @auth(user_id)
	// @sse
	// @event domain.Event
	// @input request.ChatStreamRequest
	api.POST("/chat/stream", authMiddleware, chat.ChatStream)
}
`)
	writeFile(t, root, "internal/transport/http/request/document.go", `package request

import "mime/multipart"

type UploadDocumentRequest struct {
	File *multipart.FileHeader `+"`form:\"file\" binding:\"required\"`"+`
	Tags []string `+"`form:\"tags\" binding:\"omitempty\"`"+`
}

type ChatStreamRequest struct {
	Message string `+"`json:\"message\" binding:\"required\"`"+`
}
`)
	writeFile(t, root, "internal/transport/http/response/document.go", `package response

type DocumentResponse struct {
	ID string `+"`json:\"id\"`"+`
}
`)

	if _, err := GenerateDocs(DocsOptions{Root: root}); err != nil {
		t.Fatalf("GenerateDocs error = %v", err)
	}
	var spec map[string]any
	if err := json.Unmarshal([]byte(readFile(t, root, "docs/openapi.json")), &spec); err != nil {
		t.Fatalf("openapi json invalid: %v", err)
	}
	paths := spec["paths"].(map[string]any)
	upload := paths["/api/documents"].(map[string]any)["post"].(map[string]any)
	uploadContent := upload["requestBody"].(map[string]any)["content"].(map[string]any)
	if uploadContent["multipart/form-data"] == nil {
		t.Fatalf("upload content = %+v, want multipart/form-data", uploadContent)
	}
	stream := paths["/api/chat/stream"].(map[string]any)["post"].(map[string]any)
	responses := stream["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)
	if responses["text/event-stream"] == nil {
		t.Fatalf("stream response content = %+v, want text/event-stream", responses)
	}
}

func TestRunDocsCommandCheckFailsWhenSpecOutdated(t *testing.T) {
	// 验证 docs --check 在 OpenAPI 文件缺失或过期时返回 ErrCheckFailed，供 CI 使用。
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	// routegen: output=response.BookResponse
	api.GET("/books", book.List)
}
`)
	writeFile(t, root, "internal/transport/http/response/book.go", `package response

type BookResponse struct {
	ID string `+"`json:\"id\"`"+`
}
`)

	report, err := runDocsCommand([]string{"-root", root, "--check", "--no-color"})
	if !errors.Is(err, ErrCheckFailed) {
		t.Fatalf("runDocsCommand error = %v, want ErrCheckFailed", err)
	}
	assertReportContains(t, report, FileWouldAdd, "docs/openapi.json")
}

func TestPrintReportJSONIncludesModeCommandAndDiagnostics(t *testing.T) {
	// 验证 JSON report 可被脚本解析，并包含工具模式、命令和诊断信息。
	report := Report{
		Command: "doctor",
		Mode:    ModeCheck,
		Files: []FileChange{
			{Kind: FileWouldAdd, Path: "internal/logic/book/show.go"},
		},
		Diagnostics: []Diagnostic{
			{Level: "error", Code: "route.sse.missing_event", Message: "missing @event"},
		},
	}
	var out bytes.Buffer
	if err := printReportWithFormat(&out, report, ReportFormatJSON, false); err != nil {
		t.Fatalf("printReportWithFormat error = %v", err)
	}
	var got Report
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal error = %v\n%s", err, out.String())
	}
	if got.Command != "doctor" || got.Mode != ModeCheck || len(got.Diagnostics) != 1 {
		t.Fatalf("json report = %+v", got)
	}
}

func TestListRouteItemsShowsGenerationStatus(t *testing.T) {
	// 验证 route --list 的底层数据能展示路由和 handler/logic 是否已存在。
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	// routegen: output=response.BookResponse
	api.GET("/books/:id", book.Show)
}
`)
	writeFile(t, root, "internal/transport/http/handler/book.go", `package handler

type BookHandler struct{}

func (h BookHandler) Show() {}
`)

	items, err := ListRoutes(root)
	if err != nil {
		t.Fatalf("ListRoutes error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	item := items[0]
	if item.HTTPMethod != "GET" || item.Path != "/books/:id" || !item.HandlerExists || item.LogicExists {
		t.Fatalf("route item = %+v", item)
	}
}

func TestListRepositoryModelsMarksScopeRequirement(t *testing.T) {
	// 验证 repository --list-models 能区分直接 user_id 模型和需要 scope 的模型。
	root := t.TempDir()
	writeFile(t, root, "internal/models/book.go", `package models

import "github.com/google/uuid"

type Book struct {
	ID     uuid.UUID `+"`gorm:\"column:id\"`"+`
	UserID uuid.UUID `+"`gorm:\"column:user_id\"`"+`
}
`)
	writeFile(t, root, "internal/models/message.go", `package models

import "github.com/google/uuid"

type Message struct {
	ID             uuid.UUID `+"`gorm:\"column:id\"`"+`
	ConversationID uuid.UUID `+"`gorm:\"column:conversation_id\"`"+`
}
`)

	models, err := ListRepositoryModels(root)
	if err != nil {
		t.Fatalf("ListRepositoryModels error = %v", err)
	}
	got := map[string]RepositoryModelItem{}
	for _, item := range models {
		got[item.Model] = item
	}
	if got["Book"].Scope != "direct" || got["Book"].RequiresScope {
		t.Fatalf("Book item = %+v", got["Book"])
	}
	if got["Message"].Scope != "requires_scope" || !got["Message"].RequiresScope {
		t.Fatalf("Message item = %+v", got["Message"])
	}
}

func TestDoctorReportsRouteAndRepositoryProblems(t *testing.T) {
	// 验证 doctor 只扫描诊断问题，不生成文件。
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/chat.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterChatRoutes(api *gin.RouterGroup, chat handler.ChatHandler) {
	chatRoutes := api.Group("/chat")
	// @sse
	chatRoutes.POST("/stream", chat.Stream)
}
`)
	writeFile(t, root, "internal/models/message.go", `package models

import "github.com/google/uuid"

type Message struct {
	ID             uuid.UUID `+"`gorm:\"column:id\"`"+`
	ConversationID uuid.UUID `+"`gorm:\"column:conversation_id\"`"+`
}
`)

	report, err := RunDoctor(DoctorOptions{Root: root})
	if err != nil {
		t.Fatalf("RunDoctor error = %v", err)
	}
	if !report.HasDiagnostic("route.sse.missing_event") {
		t.Fatalf("diagnostics = %+v, want missing event", report.Diagnostics)
	}
	if !report.HasDiagnostic("repository.scope.required") {
		t.Fatalf("diagnostics = %+v, want required scope", report.Diagnostics)
	}
	if len(report.Files) != 0 {
		t.Fatalf("files = %+v, want no generated files", report.Files)
	}
}

func TestDoctorReportsMissingResponseAndURIField(t *testing.T) {
	// 验证 doctor 能发现缺少响应声明，以及 path 参数没有对应 uri 字段的问题。
	root := t.TempDir()
	writeFile(t, root, "internal/transport/http/routes/book.go", `package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBookRoutes(api *gin.RouterGroup, book handler.BookHandler) {
	// @input request.RenameBookRequest
	api.PATCH("/books/:book_id", book.Rename)
}
`)
	writeFile(t, root, "internal/transport/http/request/book.go", `package request

type RenameBookRequest struct {
	Title string `+"`json:\"title\" binding:\"required\"`"+`
}
`)

	report, err := RunDoctor(DoctorOptions{Root: root})
	if err != nil {
		t.Fatalf("RunDoctor error = %v", err)
	}
	if !report.HasDiagnostic("route.response.missing") {
		t.Fatalf("diagnostics = %+v, want missing response", report.Diagnostics)
	}
	if !report.HasDiagnostic("route.uri.missing_field") {
		t.Fatalf("diagnostics = %+v, want missing uri field", report.Diagnostics)
	}
}

func TestGenerateRepositoryCreatesConversationStyleRepository(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/models/book.go", `package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID uuid.UUID `+"`gorm:\"column:id;type:uuid;primaryKey\"`"+`
}

type Book struct {
	ID        uuid.UUID `+"`gorm:\"column:id;type:uuid;primaryKey\"`"+`
	UserID    uuid.UUID `+"`gorm:\"column:user_id;type:uuid;not null;index\"`"+`
	User      User      `+"`gorm:\"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE\"`"+`
	Title     string    `+"`gorm:\"column:title;size:255;not null\"`"+`
	IsPublic  bool      `+"`gorm:\"column:is_public;not null;default:false\"`"+`
	CreatedAt time.Time `+"`gorm:\"column:created_at;autoCreateTime\"`"+`
	UpdatedAt time.Time `+"`gorm:\"column:updated_at;autoUpdateTime\"`"+`
}
`)

	report, err := GenerateRepository(RepositoryOptions{Root: root, Model: "Book", Label: "图书"})
	if err != nil {
		t.Fatalf("GenerateRepository error = %v", err)
	}

	repoFile := readFile(t, root, "internal/repository/book.go")
	for _, want := range []string{
		"// Code generated by codegen; DO NOT EDIT.",
		"type BookRepository interface",
		"Create(ctx context.Context, userID uuid.UUID, book *models.Book) (*models.Book, error)",
		"UpdateFields(ctx context.Context, userID uuid.UUID, bookID uuid.UUID, book *models.Book, fields *BookUpdateFields) (*models.Book, error)",
		"func NewBookUpdateFields() *BookUpdateFields",
		"func (f *BookUpdateFields) Title() *BookUpdateFields",
		"return f.add(\"title\")",
		"func (f *BookUpdateFields) IsPublic() *BookUpdateFields",
		"return f.add(\"is_public\")",
	} {
		if !strings.Contains(repoFile, want) {
			t.Fatalf("repository file missing %q:\n%s", want, repoFile)
		}
	}
	for _, notWant := range []string{
		"func (f *BookUpdateFields) ID()",
		"func (f *BookUpdateFields) UserID()",
		"func (f *BookUpdateFields) User()",
		"func (f *BookUpdateFields) CreatedAt()",
		"func (f *BookUpdateFields) UpdatedAt()",
	} {
		if strings.Contains(repoFile, notWant) {
			t.Fatalf("repository file should not contain %q:\n%s", notWant, repoFile)
		}
	}

	postgresFile := readFile(t, root, "internal/repository/postgres/book.go")
	for _, want := range []string{
		"type BookRepository struct",
		"func NewBookRepository(db *gorm.DB) repository.BookRepository",
		"book.UserID = userID",
		"Where(\"id = ? AND user_id = ?\", bookID, userID)",
		"Order(\"updated_at DESC\")",
		"xerr.NotFound(\"图书不存在\")",
		"xerr.Wrapf(err, \"创建图书失败\")",
		"Select(columns)",
		"Delete(&models.Book{})",
	} {
		if !strings.Contains(postgresFile, want) {
			t.Fatalf("postgres repository file missing %q:\n%s", want, postgresFile)
		}
	}

	assertReportContains(t, report, FileAdded, "internal/repository/book.go")
	assertReportContains(t, report, FileAdded, "internal/repository/postgres/book.go")
	assertReportContains(t, report, FileAdded, "internal/models/hooks.go")

	hooksFile := readFile(t, root, "internal/models/hooks.go")
	for _, want := range []string{
		"package models",
		`"github.com/google/uuid"`,
		`"gorm.io/gorm"`,
		"func ensureUUID(id *uuid.UUID)",
		"func (b *Book) BeforeCreate(tx *gorm.DB) error",
		"ensureUUID(&b.ID)",
	} {
		if !strings.Contains(hooksFile, want) {
			t.Fatalf("hooks file missing %q:\n%s", want, hooksFile)
		}
	}
}

func TestGenerateRepositoryRequiresUserScopedUUIDModel(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/models/book.go", `package models

import "github.com/google/uuid"

type Book struct {
	ID uuid.UUID `+"`gorm:\"column:id;type:uuid;primaryKey\"`"+`
}
`)

	_, err := GenerateRepository(RepositoryOptions{Root: root, Model: "Book"})
	if err == nil {
		t.Fatal("GenerateRepository error = nil, want missing UserID error")
	}
	if !strings.Contains(err.Error(), "must have UserID uuid.UUID or provide -scope local_column:table.column:user_column") {
		t.Fatalf("GenerateRepository error = %v", err)
	}
}

func TestGenerateRepositoryCreatesJoinScopedRepository(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/models/message.go", `package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID             uuid.UUID      `+"`gorm:\"column:id;type:uuid;primaryKey\"`"+`
	ConversationID uuid.UUID      `+"`gorm:\"column:conversation_id;type:uuid;not null;index\"`"+`
	Conversation   Conversation   `+"`gorm:\"foreignKey:ConversationID;references:ID;constraint:OnDelete:CASCADE\"`"+`
	Role           string         `+"`gorm:\"column:role;size:16;not null\"`"+`
	SenderPersonID uuid.UUID      `+"`gorm:\"column:sender_person_id;type:uuid\"`"+`
	SenderUserID   uuid.UUID      `+"`gorm:\"column:sender_user_id;type:uuid\"`"+`
	MetaData       map[string]any `+"`gorm:\"column:meta_data;type:jsonb\"`"+`
	CreatedAt      time.Time      `+"`gorm:\"column:created_at;autoCreateTime\"`"+`
}
`)

	report, err := GenerateRepository(RepositoryOptions{
		Root:  root,
		Model: "Message",
		Label: "消息",
		Scope: "conversation_id:conversations.id:user_id",
	})
	if err != nil {
		t.Fatalf("GenerateRepository error = %v", err)
	}

	repoFile := readFile(t, root, "internal/repository/message.go")
	for _, want := range []string{
		"type MessageRepository interface",
		"Create(ctx context.Context, userID uuid.UUID, message *models.Message) (*models.Message, error)",
		"func (f *MessageUpdateFields) Role() *MessageUpdateFields",
		"func (f *MessageUpdateFields) SenderPersonID() *MessageUpdateFields",
		"func (f *MessageUpdateFields) SenderUserID() *MessageUpdateFields",
		"func (f *MessageUpdateFields) MetaData() *MessageUpdateFields",
	} {
		if !strings.Contains(repoFile, want) {
			t.Fatalf("repository file missing %q:\n%s", want, repoFile)
		}
	}
	for _, notWant := range []string{
		"func (f *MessageUpdateFields) ID()",
		"func (f *MessageUpdateFields) Conversation()",
		"func (f *MessageUpdateFields) CreatedAt()",
	} {
		if strings.Contains(repoFile, notWant) {
			t.Fatalf("repository file should not contain %q:\n%s", notWant, repoFile)
		}
	}

	postgresFile := readFile(t, root, "internal/repository/postgres/message.go")
	for _, want := range []string{
		"Joins(\"JOIN conversations ON messages.conversation_id = conversations.id\")",
		"Where(\"conversations.user_id = ?\", userID)",
		"Where(\"messages.id = ?\", messageID)",
		"EXISTS (SELECT 1 FROM conversations WHERE conversations.id = messages.conversation_id AND conversations.user_id = ?)",
		"Where(\"id = ? AND user_id = ?\", message.ConversationID, userID)",
		"xerr.NotFound(\"消息不存在\")",
		"Order(\"created_at DESC\")",
	} {
		if !strings.Contains(postgresFile, want) {
			t.Fatalf("postgres repository file missing %q:\n%s", want, postgresFile)
		}
	}
	if strings.Contains(postgresFile, "message.UserID = userID") {
		t.Fatalf("join-scoped repository should not assign UserID:\n%s", postgresFile)
	}

	assertReportContains(t, report, FileAdded, "internal/repository/message.go")
	assertReportContains(t, report, FileAdded, "internal/repository/postgres/message.go")
	assertReportContains(t, report, FileAdded, "internal/models/hooks.go")

	hooksFile := readFile(t, root, "internal/models/hooks.go")
	for _, want := range []string{
		"func (m *Message) BeforeCreate(tx *gorm.DB) error",
		"ensureUUID(&m.ID)",
	} {
		if !strings.Contains(hooksFile, want) {
			t.Fatalf("hooks file missing %q:\n%s", want, hooksFile)
		}
	}
}

func TestGenerateRepositoryUsesAcronymAwareFilenames(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/models/mcp_server.go", `package models

import "github.com/google/uuid"

type MCPServer struct {
	ID     uuid.UUID `+"`gorm:\"column:id;type:uuid;primaryKey\"`"+`
	UserID uuid.UUID `+"`gorm:\"column:user_id;type:uuid;not null;index\"`"+`
	Name   string    `+"`gorm:\"column:name\"`"+`
}
`)

	report, err := GenerateRepository(RepositoryOptions{Root: root, Model: "MCPServer", Label: "MCP 服务"})
	if err != nil {
		t.Fatalf("GenerateRepository error = %v", err)
	}

	assertReportContains(t, report, FileAdded, "internal/repository/mcp_server.go")
	assertReportContains(t, report, FileAdded, "internal/repository/postgres/mcp_server.go")
	repoFile := readFile(t, root, "internal/repository/mcp_server.go")
	if !strings.Contains(repoFile, "type MCPServerRepository interface") {
		t.Fatalf("repository file missing MCPServerRepository:\n%s", repoFile)
	}
	for _, path := range []string{
		"internal/repository/m_c_p_server.go",
		"internal/repository/postgres/m_c_p_server.go",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err == nil {
			t.Fatalf("generated old acronym-split repository file %s", path)
		}
	}
}

func TestGenerateRepositorySkipsExistingFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/models/book.go", `package models

import "github.com/google/uuid"

type Book struct {
	ID     uuid.UUID `+"`gorm:\"column:id;type:uuid;primaryKey\"`"+`
	UserID uuid.UUID `+"`gorm:\"column:user_id;type:uuid;not null;index\"`"+`
	Title  string    `+"`gorm:\"column:title\"`"+`
}
`)
	existingRepo := "package repository\n\n// custom repository\n"
	existingPostgres := "package postgres\n\n// custom postgres repository\n"
	writeFile(t, root, "internal/repository/book.go", existingRepo)
	writeFile(t, root, "internal/repository/postgres/book.go", existingPostgres)

	report, err := GenerateRepository(RepositoryOptions{Root: root, Model: "Book"})
	if err != nil {
		t.Fatalf("GenerateRepository error = %v", err)
	}
	if got := readFile(t, root, "internal/repository/book.go"); got != existingRepo {
		t.Fatalf("existing repository changed:\n%s", got)
	}
	if got := readFile(t, root, "internal/repository/postgres/book.go"); got != existingPostgres {
		t.Fatalf("existing postgres repository changed:\n%s", got)
	}
	assertReportContains(t, report, FileSkipped, "internal/repository/book.go")
	assertReportContains(t, report, FileSkipped, "internal/repository/postgres/book.go")
}

func TestGenerateRepositoryAppendsMissingHook(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/models/book.go", `package models

import "github.com/google/uuid"

type Book struct {
	ID     uuid.UUID `+"`gorm:\"column:id;type:uuid;primaryKey\"`"+`
	UserID uuid.UUID `+"`gorm:\"column:user_id;type:uuid;not null;index\"`"+`
	Title  string    `+"`gorm:\"column:title\"`"+`
}
`)
	writeFile(t, root, "internal/models/hooks.go", `package models

func existingHelper() {}
`)

	report, err := GenerateRepository(RepositoryOptions{Root: root, Model: "Book"})
	if err != nil {
		t.Fatalf("GenerateRepository error = %v", err)
	}

	hooksFile := readFile(t, root, "internal/models/hooks.go")
	for _, want := range []string{
		`"github.com/google/uuid"`,
		`"gorm.io/gorm"`,
		"func existingHelper() {}",
		"func ensureUUID(id *uuid.UUID)",
		"func (b *Book) BeforeCreate(tx *gorm.DB) error",
		"ensureUUID(&b.ID)",
	} {
		if !strings.Contains(hooksFile, want) {
			t.Fatalf("hooks file missing %q:\n%s", want, hooksFile)
		}
	}
	assertReportContains(t, report, FileModified, "internal/models/hooks.go")
}

func TestGenerateRepositorySkipsExistingHook(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/models/book.go", `package models

import "github.com/google/uuid"

type Book struct {
	ID     uuid.UUID `+"`gorm:\"column:id;type:uuid;primaryKey\"`"+`
	UserID uuid.UUID `+"`gorm:\"column:user_id;type:uuid;not null;index\"`"+`
	Title  string    `+"`gorm:\"column:title\"`"+`
}
`)
	writeFile(t, root, "internal/models/hooks.go", `package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func ensureUUID(id *uuid.UUID) {
	if *id == uuid.Nil {
		*id = uuid.New()
	}
}

func (b *Book) BeforeCreate(tx *gorm.DB) error {
	ensureUUID(&b.ID)
	return nil
}
`)

	report, err := GenerateRepository(RepositoryOptions{Root: root, Model: "Book"})
	if err != nil {
		t.Fatalf("GenerateRepository error = %v", err)
	}

	hooksFile := readFile(t, root, "internal/models/hooks.go")
	if strings.Count(hooksFile, "BeforeCreate") != 1 {
		t.Fatalf("hooks file should contain one BeforeCreate:\n%s", hooksFile)
	}
	assertReportContains(t, report, FileSkipped, "internal/models/hooks.go")
}

func assertReportContains(t *testing.T, report Report, kind FileChangeKind, path string) {
	t.Helper()
	if !report.Has(kind, path) {
		t.Fatalf("report = %+v, want %s %s", report, kind, path)
	}
}

func writeFile(t *testing.T, root, path, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll error = %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
}

func readFile(t *testing.T, root, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	return string(data)
}

func commentGroup(lines ...string) *ast.CommentGroup {
	comments := make([]*ast.Comment, 0, len(lines))
	for _, line := range lines {
		comments = append(comments, &ast.Comment{Text: line})
	}
	return &ast.CommentGroup{List: comments}
}
