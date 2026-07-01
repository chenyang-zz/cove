package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

const defaultDocsOutput = "docs/openapi.json"

type docDTO struct {
	Name   string
	Fields []docField
}

type docField struct {
	Name      string
	Type      string
	JSONName  string
	FormName  string
	QueryName string
	URIName   string
	Binding   string
	Embedded  string
	File      bool
}

func GenerateDocs(opts DocsOptions) (Report, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}
	report := Report{Root: root, Command: "docs", Mode: generationMode(opts.DryRun, opts.Check)}
	if opts.DryRun && opts.Check {
		return report, fmt.Errorf("--dry-run and --check are mutually exclusive")
	}
	if opts.Output == "" {
		opts.Output = defaultDocsOutput
	}
	if opts.Title == "" {
		opts.Title = "Boxify API"
	}
	if opts.Version == "" {
		opts.Version = "0.1.0"
	}

	routes, err := scanRoutes(root)
	if err != nil {
		return report, err
	}
	if err := validateRoutes(routes); err != nil {
		return report, err
	}
	requests, err := scanDocDTOs(filepath.Join(root, "internal", "transport", "http", "request"))
	if err != nil {
		return report, err
	}
	responses, err := scanDocDTOs(filepath.Join(root, "internal", "transport", "http", "response"))
	if err != nil {
		return report, err
	}
	spec, err := buildOpenAPISpec(routes, requests, responses, opts.Title, opts.Version)
	if err != nil {
		return report, err
	}
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return report, err
	}
	data = append(data, '\n')
	if err := validateOpenAPISpec(data); err != nil {
		return report, err
	}
	if opts.Verbose {
		report.AddDiagnostic("info", "docs.routes_scanned", fmt.Sprintf("generated OpenAPI spec for %d routes", len(routes)), "", "")
	}
	if err := writeDataFile(filepath.Join(root, filepath.FromSlash(opts.Output)), data, &report); err != nil {
		return report, err
	}
	return report, nil
}

func scanDocDTOs(dir string) (map[string]docDTO, error) {
	out := map[string]docDTO{}
	err := scanGoFiles(dir, func(path string, file *ast.File) {
		fset := token.NewFileSet()
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				out[typeSpec.Name.Name] = docDTO{Name: typeSpec.Name.Name, Fields: docFields(fset, structType)}
			}
		}
	})
	return out, err
}

func docFields(fset *token.FileSet, structType *ast.StructType) []docField {
	if structType == nil || structType.Fields == nil {
		return nil
	}
	fields := make([]docField, 0, len(structType.Fields.List))
	for _, field := range structType.Fields.List {
		item := docField{Type: exprString(fset, field.Type)}
		if field.Tag != nil {
			item.JSONName = tagName(docTagValue(field.Tag.Value, "json"))
			item.FormName = tagName(docTagValue(field.Tag.Value, "form"))
			item.QueryName = tagName(docTagValue(field.Tag.Value, "query"))
			item.URIName = tagName(docTagValue(field.Tag.Value, "uri"))
			item.Binding = docTagValue(field.Tag.Value, "binding")
		}
		if len(field.Names) == 0 {
			item.Embedded = requestEmbeddedTypeName(field.Type)
		} else {
			item.Name = field.Names[0].Name
		}
		item.File = item.FormName != "" && requestExprIsMultipartFileHeader(field.Type)
		fields = append(fields, item)
	}
	return fields
}

func docTagValue(raw string, key string) string {
	if raw == "" {
		return ""
	}
	unquoted, err := strconv.Unquote(raw)
	if err != nil {
		return ""
	}
	return reflect.StructTag(unquoted).Get(key)
}

func buildOpenAPISpec(routes []Route, requests map[string]docDTO, responses map[string]docDTO, title string, version string) (map[string]any, error) {
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].HTTPMethod < routes[j].HTTPMethod
		}
		return routes[i].Path < routes[j].Path
	})

	componentsSchemas := map[string]any{
		"ErrorEnvelope": map[string]any{
			"type":     "object",
			"required": []string{"code", "message"},
			"properties": map[string]any{
				"code":    map[string]any{"type": "integer"},
				"message": map[string]any{"type": "string"},
			},
		},
	}
	paths := map[string]any{}
	for _, route := range routes {
		openAPIPath := ginPathToOpenAPI(joinRoutePath("/api", route.Path))
		pathItem, _ := paths[openAPIPath].(map[string]any)
		if pathItem == nil {
			pathItem = map[string]any{}
			paths[openAPIPath] = pathItem
		}
		operation, err := routeOperation(route, requests, responses, componentsSchemas)
		if err != nil {
			return nil, err
		}
		pathItem[strings.ToLower(route.HTTPMethod)] = operation
	}
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   title,
			"version": version,
		},
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"BearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
			"schemas": componentsSchemas,
		},
	}, nil
}

func routeOperation(route Route, requests map[string]docDTO, responses map[string]docDTO, components map[string]any) (map[string]any, error) {
	op := map[string]any{
		"operationId": lowerFirst(route.Domain) + route.HandlerMethod,
		"tags":        []string{routeTag(route)},
		"responses": map[string]any{
			"400": map[string]any{"description": "请求参数错误", "content": jsonContent(schemaRef("ErrorEnvelope"))},
			"401": map[string]any{"description": "未认证", "content": jsonContent(schemaRef("ErrorEnvelope"))},
			"500": map[string]any{"description": "服务器内部错误", "content": jsonContent(schemaRef("ErrorEnvelope"))},
		},
	}
	if route.Directive.Summary != "" {
		op["summary"] = route.Directive.Summary
	}
	description := strings.Join(route.CommentLines, "\n")
	if len(route.Directive.Description) > 0 {
		description = strings.Join(route.Directive.Description, "\n")
	}
	if description != "" {
		op["description"] = description
	}
	if route.Directive.Auth {
		op["security"] = []any{map[string]any{"BearerAuth": []string{}}}
	}

	var input *docDTO
	if route.Directive.Input != "" {
		name := requestTypeName(route.Directive.Input)
		dto, ok := requests[name]
		if !ok {
			return nil, fmt.Errorf("docs: request DTO %s not found for %s.%s", route.Directive.Input, route.HandlerType, route.HandlerMethod)
		}
		input = &dto
		params := parametersForRoute(route, dto, requests)
		if len(params) > 0 {
			op["parameters"] = params
		}
		if route.HTTPMethod != "GET" {
			if requestHasMultipart(dto, requests, map[string]bool{}) {
				op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"multipart/form-data": map[string]any{"schema": bodySchema(dto, requests, components, true)}}}
			} else if requestHasJSON(dto, requests, map[string]bool{}) {
				op["requestBody"] = map[string]any{"required": true, "content": jsonContent(bodySchema(dto, requests, components, false))}
			}
		}
	}

	success := map[string]any{"description": "ok"}
	if route.Directive.SSE {
		success["content"] = map[string]any{"text/event-stream": map[string]any{"schema": map[string]any{"type": "string"}}}
	} else if route.Directive.Output != "" {
		success["content"] = jsonContent(envelopeSchema(typeSchema(route.Directive.Output, responses, components)))
	} else if input != nil {
		success["content"] = jsonContent(envelopeSchema(nil))
	}
	op["responses"].(map[string]any)["200"] = success
	return op, nil
}

func routeTag(route Route) string {
	if route.Directive.Tag != "" {
		return route.Directive.Tag
	}
	return route.Domain
}

func parametersForRoute(route Route, dto docDTO, requests map[string]docDTO) []any {
	var params []any
	declaredPath := map[string]struct{}{}
	for _, field := range flattenDocFields(dto, requests, map[string]bool{}) {
		switch {
		case field.URIName != "":
			declaredPath[field.URIName] = struct{}{}
			params = append(params, parameter(field.URIName, "path", true, field))
		case route.HTTPMethod == "GET" && formOrQueryName(field) != "":
			params = append(params, parameter(formOrQueryName(field), "query", bindingHas(field.Binding, "required"), field))
		}
	}
	for _, name := range routeURIParamNames(route.Path) {
		if _, ok := declaredPath[name]; ok {
			continue
		}
		params = append(params, map[string]any{
			"name":     name,
			"in":       "path",
			"required": true,
			"schema":   map[string]any{"type": "string"},
		})
	}
	return params
}

func parameter(name string, in string, required bool, field docField) map[string]any {
	if in == "path" {
		required = true
	}
	return map[string]any{
		"name":     name,
		"in":       in,
		"required": required,
		"schema":   schemaForField(field),
	}
}

func bodySchema(dto docDTO, requests map[string]docDTO, components map[string]any, multipart bool) map[string]any {
	required := []string{}
	properties := map[string]any{}
	for _, field := range flattenDocFields(dto, requests, map[string]bool{}) {
		name := field.JSONName
		if multipart {
			name = formOrQueryName(field)
		}
		if name == "" || field.URIName != "" {
			continue
		}
		if multipart && field.JSONName != "" && field.FormName == "" && field.QueryName == "" {
			continue
		}
		properties[name] = schemaForField(field)
		if bindingHas(field.Binding, "required") {
			required = append(required, name)
		}
	}
	sort.Strings(required)
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func flattenDocFields(dto docDTO, all map[string]docDTO, visiting map[string]bool) []docField {
	if visiting[dto.Name] {
		return nil
	}
	visiting[dto.Name] = true
	defer delete(visiting, dto.Name)

	var out []docField
	for _, field := range dto.Fields {
		if field.Embedded != "" {
			if embedded, ok := all[field.Embedded]; ok {
				out = append(out, flattenDocFields(embedded, all, visiting)...)
			}
			continue
		}
		out = append(out, field)
	}
	return out
}

func requestHasJSON(dto docDTO, all map[string]docDTO, visiting map[string]bool) bool {
	for _, field := range flattenDocFields(dto, all, visiting) {
		if field.JSONName != "" && field.URIName == "" {
			return true
		}
	}
	return false
}

func requestHasMultipart(dto docDTO, all map[string]docDTO, visiting map[string]bool) bool {
	for _, field := range flattenDocFields(dto, all, visiting) {
		if field.File {
			return true
		}
	}
	return false
}

func formOrQueryName(field docField) string {
	if field.FormName != "" {
		return field.FormName
	}
	return field.QueryName
}

func schemaForField(field docField) map[string]any {
	schema := schemaForType(field.Type)
	if field.File {
		return map[string]any{"type": "string", "format": "binary"}
	}
	for _, rule := range strings.Split(field.Binding, ",") {
		rule = strings.TrimSpace(rule)
		switch {
		case strings.HasPrefix(rule, "min="):
			if v, err := strconv.Atoi(strings.TrimPrefix(rule, "min=")); err == nil {
				if schema["type"] == "string" {
					schema["minLength"] = v
				} else {
					schema["minimum"] = v
				}
			}
		case strings.HasPrefix(rule, "max="):
			if v, err := strconv.Atoi(strings.TrimPrefix(rule, "max=")); err == nil {
				if schema["type"] == "string" {
					schema["maxLength"] = v
				} else {
					schema["maximum"] = v
				}
			}
		case strings.HasPrefix(rule, "oneof="):
			values := strings.Fields(strings.TrimPrefix(rule, "oneof="))
			if len(values) > 0 {
				schema["enum"] = values
			}
		case rule == "email":
			schema["format"] = "email"
		}
	}
	return schema
}

func schemaForType(typeName string) map[string]any {
	typeName = strings.TrimPrefix(strings.TrimSpace(typeName), "*")
	switch {
	case strings.HasPrefix(typeName, "[]"):
		return map[string]any{"type": "array", "items": schemaForType(strings.TrimPrefix(typeName, "[]"))}
	case typeName == "string" || typeName == "uuid.UUID":
		return map[string]any{"type": "string"}
	case strings.HasPrefix(typeName, "int") || strings.HasPrefix(typeName, "uint"):
		return map[string]any{"type": "integer"}
	case strings.HasPrefix(typeName, "float"):
		return map[string]any{"type": "number"}
	case typeName == "bool":
		return map[string]any{"type": "boolean"}
	case typeName == "time.Time":
		return map[string]any{"type": "string", "format": "date-time"}
	default:
		return map[string]any{"type": "object"}
	}
}

func typeSchema(typeExpr string, responses map[string]docDTO, components map[string]any) map[string]any {
	typeExpr = strings.TrimSpace(typeExpr)
	for _, prefix := range []string{"*", "[]*"} {
		if strings.HasPrefix(typeExpr, prefix) {
			inner := strings.TrimPrefix(typeExpr, prefix)
			if prefix == "[]*" {
				return map[string]any{"type": "array", "items": typeSchema(inner, responses, components)}
			}
			return typeSchema(inner, responses, components)
		}
	}
	if strings.HasPrefix(typeExpr, "[]") {
		return map[string]any{"type": "array", "items": typeSchema(strings.TrimPrefix(typeExpr, "[]"), responses, components)}
	}
	if strings.HasPrefix(typeExpr, "response.ListResponse[") || strings.HasPrefix(typeExpr, "ListResponse[") {
		inner := genericInner(typeExpr)
		return map[string]any{"type": "object", "properties": map[string]any{"list": map[string]any{"type": "array", "items": typeSchema(inner, responses, components)}}}
	}
	if strings.HasPrefix(typeExpr, "response.PageListResponse[") || strings.HasPrefix(typeExpr, "PageListResponse[") {
		inner := genericInner(typeExpr)
		return map[string]any{"type": "object", "properties": map[string]any{
			"total":     map[string]any{"type": "integer"},
			"page":      map[string]any{"type": "integer"},
			"page_size": map[string]any{"type": "integer"},
			"list":      map[string]any{"type": "array", "items": typeSchema(inner, responses, components)},
		}}
	}
	name := typeRefName(typeExpr)
	if name == "" {
		return map[string]any{"type": "object"}
	}
	if dto, ok := responses[name]; ok {
		if _, exists := components[name]; !exists {
			components[name] = objectSchema(dto, responses, components)
		}
		return schemaRef(name)
	}
	return schemaForType(name)
}

func objectSchema(dto docDTO, responses map[string]docDTO, components map[string]any) map[string]any {
	properties := map[string]any{}
	required := []string{}
	for _, field := range flattenDocFields(dto, responses, map[string]bool{}) {
		if field.JSONName == "" {
			continue
		}
		properties[field.JSONName] = schemaForField(field)
		if bindingHas(field.Binding, "required") {
			required = append(required, field.JSONName)
		}
	}
	sort.Strings(required)
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func genericInner(typeExpr string) string {
	start := strings.Index(typeExpr, "[")
	end := strings.LastIndex(typeExpr, "]")
	if start < 0 || end <= start {
		return ""
	}
	return strings.TrimSpace(typeExpr[start+1 : end])
}

func typeRefName(typeExpr string) string {
	typeExpr = strings.TrimSpace(typeExpr)
	typeExpr = strings.TrimPrefix(typeExpr, "*")
	if strings.Contains(typeExpr, "[") {
		return ""
	}
	if idx := strings.LastIndex(typeExpr, "."); idx >= 0 {
		return typeExpr[idx+1:]
	}
	return typeExpr
}

func envelopeSchema(data map[string]any) map[string]any {
	properties := map[string]any{
		"code":    map[string]any{"type": "integer"},
		"message": map[string]any{"type": "string"},
	}
	if data != nil {
		properties["data"] = data
	}
	return map[string]any{
		"type":       "object",
		"required":   []string{"code", "message"},
		"properties": properties,
	}
}

func jsonContent(schema map[string]any) map[string]any {
	return map[string]any{"application/json": map[string]any{"schema": schema}}
}

func schemaRef(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func bindingHas(binding string, rule string) bool {
	for _, item := range strings.Split(binding, ",") {
		if strings.TrimSpace(item) == rule {
			return true
		}
	}
	return false
}

func ginPathToOpenAPI(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") && len(part) > 1 {
			parts[i] = "{" + strings.TrimPrefix(part, ":") + "}"
		}
	}
	return strings.Join(parts, "/")
}

func validateOpenAPISpec(data []byte) error {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return fmt.Errorf("load generated openapi: %w", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return fmt.Errorf("validate generated openapi: %w", err)
	}
	return nil
}
