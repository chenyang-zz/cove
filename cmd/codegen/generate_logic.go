package main

import (
	"fmt"
	"sort"
	"strings"
)

func generateLogic(root string, route Route, report *Report) error {
	methodArgs, imports := logicMethodArgs(route)
	output := route.Directive.Output
	if route.Directive.SSE {
		output = "<-chan " + strings.TrimSpace(route.Directive.Event)
	}
	body, err := renderTemplate("logic.gotmpl", map[string]any{
		"LogicType":       route.HandlerMethod + "Logic",
		"Method":          route.HandlerMethod,
		"UseCase":         lowerFirst(route.HandlerMethod),
		"Domain":          route.Domain,
		"Component":       strings.ToLower(route.HandlerMethod),
		"Args":            strings.Join(methodArgs, ", "),
		"ReturnSignature": logicReturnSignature(output),
		"ZeroReturn":      logicZeroReturn(output),
		"MethodComment":   logicMethodComment(route),
	})
	if err != nil {
		return err
	}

	imports = append(imports,
		`"context"`,
		`"log/slog"`,
		fmt.Sprintf(`"%s/internal/observability/xlog"`, modulePath),
		fmt.Sprintf(`"%s/internal/svc"`, modulePath),
	)
	sort.Strings(imports)
	content := generatedFile(route.Domain, imports, body, false)
	return writeGeneratedFile(logicPath(root, route), content, report)
}

func logicMethodComment(route Route) string {
	lines := nonEmptyLines(route.CommentLines)
	if len(lines) == 0 {
		return fmt.Sprintf("// %s handles the %s use case.", route.HandlerMethod, lowerFirst(route.HandlerMethod))
	}
	if !commentStartsWithMethod(lines[0], route.HandlerMethod) {
		lines[0] = route.HandlerMethod + " " + lines[0]
	}
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("// ")
		b.WriteString(line)
	}
	return b.String()
}

func nonEmptyLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func commentStartsWithMethod(line, method string) bool {
	line = strings.TrimSpace(line)
	return line == method || strings.HasPrefix(line, method+" ") || strings.HasPrefix(line, method+"\t")
}

func logicMethodArgs(route Route) ([]string, []string) {
	var args []string
	var imports []string
	if route.Directive.UserID {
		args = append(args, "userID uuid.UUID")
		imports = append(imports, `"github.com/google/uuid"`)
	}
	if route.Directive.Input != "" {
		args = append(args, "input *"+route.Directive.Input)
		imports = append(imports, fmt.Sprintf(`"%s/internal/transport/http/request"`, modulePath))
	}
	if route.Directive.SSE {
		imports = append(imports, logicImportsForType(route.Directive.Event)...)
		return args, imports
	}
	if strings.Contains(route.Directive.Output, "response.") {
		imports = append(imports, fmt.Sprintf(`"%s/internal/transport/http/response"`, modulePath))
	}
	return args, imports
}

func logicImportsForType(goType string) []string {
	var imports []string
	if strings.Contains(goType, "domain.") {
		imports = append(imports, fmt.Sprintf(`"%s/internal/domain"`, modulePath))
	}
	if strings.Contains(goType, "response.") {
		imports = append(imports, fmt.Sprintf(`"%s/internal/transport/http/response"`, modulePath))
	}
	return imports
}

func logicReturnSignature(goType string) string {
	goType = strings.TrimSpace(goType)
	if goType == "" {
		return "error"
	}
	return fmt.Sprintf("(%s, error)", logicReturnType(goType))
}

func logicReturnType(goType string) string {
	goType = strings.TrimSpace(goType)
	switch {
	case goType == "" || goType == "any":
		return "any"
	case strings.HasPrefix(goType, "*") || strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[") || strings.HasPrefix(goType, "<-chan ") || strings.HasPrefix(goType, "chan "):
		return goType
	default:
		return "*" + goType
	}
}

func logicZeroReturn(goType string) string {
	goType = strings.TrimSpace(goType)
	if goType == "" {
		return "\treturn nil"
	}
	return zeroReturn(logicReturnType(goType))
}

func zeroReturn(goType string) string {
	goType = strings.TrimSpace(goType)
	switch {
	case goType == "" || goType == "any":
		return "\treturn nil, nil"
	case strings.HasPrefix(goType, "*") || strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "<-chan ") || strings.HasPrefix(goType, "chan "):
		return "\treturn nil, nil"
	case goType == "string":
		return "\treturn \"\", nil"
	case goType == "bool":
		return "\treturn false, nil"
	case isNumberType(goType):
		return "\treturn 0, nil"
	default:
		return fmt.Sprintf("\treturn %s{}, nil", goType)
	}
}

func isNumberType(goType string) bool {
	switch goType {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return true
	default:
		return false
	}
}
