package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func scanRoutes(root string) ([]Route, error) {
	routesDir := filepath.Join(root, "internal", "transport", "http", "routes")
	entries, err := os.ReadDir(routesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var routes []Route
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(routesDir, entry.Name())
		fileRoutes, err := scanRouteFile(path)
		if err != nil {
			return nil, err
		}
		routes = append(routes, fileRoutes...)
	}
	return routes, nil
}

func scanRouteFile(path string) ([]Route, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var routes []Route
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		handlerTypes := handlerTypesByParam(fn)
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			method, ok := httpMethodFromCall(call)
			if !ok || len(call.Args) < 2 {
				return true
			}
			handlerSelector, handlerIdent, ok := handlerFromRouteArgs(call.Args[1:], handlerTypes)
			if !ok {
				return true
			}
			handlerType := handlerTypes[handlerIdent.Name]
			if handlerType == "" {
				return true
			}
			directive, commentLines, ok := directiveForCall(fset, file, call)
			if !ok {
				return true
			}
			routes = append(routes, Route{
				HTTPMethod:    method,
				Path:          routePathFromCall(call),
				HandlerVar:    handlerIdent.Name,
				HandlerType:   handlerType,
				HandlerMethod: handlerSelector.Sel.Name,
				Domain:        domainFromHandlerType(handlerType),
				CommentLines:  commentLines,
				Directive:     directive,
			})
			return true
		})
	}
	return routes, nil
}

func routePathFromCall(call *ast.CallExpr) string {
	if len(call.Args) == 0 {
		return ""
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return value
}

func handlerFromRouteArgs(args []ast.Expr, handlerTypes map[string]string) (*ast.SelectorExpr, *ast.Ident, bool) {
	for _, arg := range args {
		selector, ok := arg.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			continue
		}
		if handlerTypes[ident.Name] == "" {
			continue
		}
		return selector, ident, true
	}
	return nil, nil, false
}

func handlerTypesByParam(fn *ast.FuncDecl) map[string]string {
	out := map[string]string{}
	if fn.Type.Params == nil {
		return out
	}
	for _, param := range fn.Type.Params.List {
		typeName := selectorTypeName(param.Type, "handler")
		if typeName == "" {
			continue
		}
		for _, name := range param.Names {
			out[name.Name] = typeName
		}
	}
	return out
}

func selectorTypeName(expr ast.Expr, pkg string) string {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok || ident.Name != pkg {
		return ""
	}
	return selector.Sel.Name
}

func httpMethodFromCall(call *ast.CallExpr) (string, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	method := selector.Sel.Name
	_, ok = httpMethods[method]
	return method, ok
}
