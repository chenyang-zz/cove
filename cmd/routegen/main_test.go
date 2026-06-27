package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDirective(t *testing.T) {
	directive, ok := parseDirective("// routegen: auth user_id input=request.CreateBookRequest output=response.BookResponse")
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
	if routes[0].Path != "/:id" {
		t.Fatalf("route path = %q, want /:id", routes[0].Path)
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

	report, err := Generate(root)
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
		"type CreateLogic struct",
		`xlog.Component("logic.book.create")`,
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

	if _, err := Generate(root); err != nil {
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

	if _, err := Generate(root); err != nil {
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

	if _, err := Generate(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"var body request.UpdateBookRequest",
		"c.ShouldBindUri(&body)",
		"c.ShouldBindJSON(&body)",
		"booklogic.NewUpdateLogic(c.Request.Context(), h.svc).Update(userID, &body)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Index(handlerFile, "ShouldBindUri(&body)") > strings.Index(handlerFile, "ShouldBindJSON(&body)") {
		t.Fatalf("URI binding should be generated before JSON binding:\n%s", handlerFile)
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

	if _, err := Generate(root); err != nil {
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

	if _, err := Generate(root); err != nil {
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

	if _, err := Generate(root); err != nil {
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

	if _, err := Generate(root); err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	handlerFile := readFile(t, root, "internal/transport/http/handler/book.go")
	for _, want := range []string{
		"var body request.UploadBookCoverRequest",
		"c.ShouldBindUri(&body)",
		"c.ShouldBind(&body)",
		"booklogic.NewUploadCoverLogic(c.Request.Context(), h.svc).UploadCover(userID, &body)",
	} {
		if !strings.Contains(handlerFile, want) {
			t.Fatalf("handler file missing %q:\n%s", want, handlerFile)
		}
	}
	if strings.Index(handlerFile, "ShouldBindUri(&body)") > strings.Index(handlerFile, "ShouldBind(&body)") {
		t.Fatalf("URI binding should be generated before multipart binding:\n%s", handlerFile)
	}
	if strings.Contains(handlerFile, "ShouldBindJSON") {
		t.Fatalf("multipart input should not bind JSON:\n%s", handlerFile)
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

	if _, err := Generate(root); err != nil {
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

	if _, err := Generate(root); err != nil {
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

	if _, err := Generate(root); err != nil {
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

	report, err := Generate(root)
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

	report, err := Generate(root)
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

	if _, err := Generate(root); err != nil {
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

	report, err := Generate(root)
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}
	if !report.Has(FileAdded, "internal/transport/http/handler/book.go") {
		t.Fatalf("first report = %+v, want added handler", report)
	}

	report, err = Generate(root)
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
