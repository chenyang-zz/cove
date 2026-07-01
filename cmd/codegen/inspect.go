package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

func ListRoutes(root string) ([]RouteListItem, error) {
	routes, err := scanRoutes(root)
	if err != nil {
		return nil, err
	}
	handlers, err := scanHandlers(root)
	if err != nil {
		return nil, err
	}
	logics, err := scanLogics(root)
	if err != nil {
		return nil, err
	}

	items := make([]RouteListItem, 0, len(routes))
	for _, route := range routes {
		logicPathValue := logicPath(root, route)
		item := RouteListItem{
			HTTPMethod:    route.HTTPMethod,
			Path:          route.Path,
			Handler:       route.HandlerType + "." + route.HandlerMethod,
			Input:         route.Directive.Input,
			Output:        route.Directive.Output,
			Event:         route.Directive.Event,
			SSE:           route.Directive.SSE,
			HandlerExists: handlers[route.HandlerType+"."+route.HandlerMethod] != "" || handlers[route.HandlerType] != "",
			LogicExists:   logics[logicKey(route.Domain, route.HandlerMethod)] != "" || fileExists(logicPathValue),
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Handler == items[j].Handler {
			return items[i].Path < items[j].Path
		}
		return items[i].Handler < items[j].Handler
	})
	return items, nil
}

func printRouteList(w io.Writer, items []RouteListItem, format ReportFormat, color bool) error {
	switch format {
	case "", ReportFormatText:
		fmt.Fprintln(w, "codegen routes:")
		if len(items) == 0 {
			fmt.Fprintln(w, colorize("  = no routes found", ansiGray, color))
			return nil
		}
		for _, item := range items {
			state := "missing"
			if item.HandlerExists && item.LogicExists {
				state = "ready"
			}
			fmt.Fprintf(w, "  %s %s -> %s [%s]\n", item.HTTPMethod, item.Path, item.Handler, state)
		}
		return nil
	case ReportFormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	default:
		return fmt.Errorf("unsupported report format %q, want text or json", format)
	}
}

func ListRepositoryModels(root string) ([]RepositoryModelItem, error) {
	models, err := scanModels(root)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(models))
	for name := range models {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]RepositoryModelItem, 0, len(names))
	for _, name := range names {
		info := models[name]
		if !modelHasGORMColumns(info) {
			continue
		}
		item := RepositoryModelItem{
			Model:     name,
			HasID:     info.HasID,
			HasUserID: info.HasUserID,
		}
		switch {
		case info.HasID && info.HasUserID:
			item.Scope = "direct"
		case info.HasID && hasRepositoryScopeCandidate(info):
			item.Scope = "requires_scope"
			item.RequiresScope = true
			item.SuggestedScope = "local_column:table.column:user_column"
		default:
			item.Scope = "unsupported"
		}
		items = append(items, item)
	}
	return items, nil
}

func modelHasGORMColumns(info ModelInfo) bool {
	for _, field := range info.Fields {
		if field.Column != "" {
			return true
		}
	}
	return false
}

func hasRepositoryScopeCandidate(info ModelInfo) bool {
	for _, field := range info.Fields {
		if field.Column == "" || field.Column == "id" || field.Column == "user_id" {
			continue
		}
		if field.Type == "uuid.UUID" && strings.HasSuffix(field.Column, "_id") {
			return true
		}
	}
	return false
}

func printRepositoryModelList(w io.Writer, items []RepositoryModelItem, format ReportFormat, color bool) error {
	switch format {
	case "", ReportFormatText:
		fmt.Fprintln(w, "codegen repository models:")
		if len(items) == 0 {
			fmt.Fprintln(w, colorize("  = no models found", ansiGray, color))
			return nil
		}
		for _, item := range items {
			line := fmt.Sprintf("%s [%s]", item.Model, item.Scope)
			if item.RequiresScope {
				line += " use -scope local_column:table.column:user_column"
			}
			fmt.Fprintf(w, "  %s\n", colorize(line, repositoryModelColor(item, color), color))
		}
		return nil
	case ReportFormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	default:
		return fmt.Errorf("unsupported report format %q, want text or json", format)
	}
}

func repositoryModelColor(item RepositoryModelItem, color bool) string {
	if !color {
		return ""
	}
	if item.Scope == "direct" {
		return ansiGreen
	}
	if item.RequiresScope {
		return ansiYellow
	}
	return ansiGray
}

func RunDoctor(opts DoctorOptions) (Report, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}
	report := Report{Root: root, Command: "doctor", Mode: ModeCheck}

	routes, err := scanRoutes(root)
	if err != nil {
		return report, err
	}
	if opts.Verbose {
		report.AddDiagnostic("info", "doctor.routes_scanned", fmt.Sprintf("scanned %d route directives", len(routes)), "", "")
	}
	requestDTOs, err := scanRequestDTOs(root)
	if err != nil {
		return report, err
	}
	for _, route := range routes {
		if route.Directive.SSE && route.Directive.Event == "" {
			report.AddDiagnostic("error", "route.sse.missing_event", fmt.Sprintf("%s.%s uses @sse but missing @event <GoType>", route.HandlerType, route.HandlerMethod), "add @event domain.AgentEvent or another SSE event type", "")
		}
		if route.Directive.Input != "" {
			typeName := requestTypeName(route.Directive.Input)
			dto, ok := requestDTOs[typeName]
			if !ok {
				report.AddDiagnostic("warn", "route.input.unknown_dto", fmt.Sprintf("%s.%s input %s cannot be resolved", route.HandlerType, route.HandlerMethod, route.Directive.Input), "define the request DTO under internal/transport/http/request", "")
				continue
			}
			if routeHasURIParam(route) && len(dto.EmbeddedURIOnly) > 1 {
				report.AddDiagnostic("error", "route.uri.multiple_embedded", fmt.Sprintf("%s.%s input %s has multiple embedded URI-only DTOs", route.HandlerType, route.HandlerMethod, route.Directive.Input), "merge URI params into a single URI request DTO", "")
			}
		}
	}

	models, err := ListRepositoryModels(root)
	if err != nil {
		return report, err
	}
	if opts.Verbose {
		report.AddDiagnostic("info", "doctor.models_scanned", fmt.Sprintf("scanned %d GORM models", len(models)), "", "")
	}
	for _, item := range models {
		if !item.HasID {
			report.AddDiagnostic("error", "repository.id.required", fmt.Sprintf("model %s must have ID uuid.UUID to generate repository", item.Model), "add ID uuid.UUID or skip repository generation for this model", "")
			continue
		}
		if item.RequiresScope {
			report.AddDiagnostic("warn", "repository.scope.required", fmt.Sprintf("model %s must provide -scope because it has no UserID uuid.UUID", item.Model), "use -scope local_column:table.column:user_column", "")
		}
	}
	return report, nil
}
