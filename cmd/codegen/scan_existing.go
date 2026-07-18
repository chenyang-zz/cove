package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func logicFileExists(root string, route Route) bool {
	_, err := os.Stat(logicPath(root, route))
	return err == nil
}

func logicPath(root string, route Route) string {
	return filepath.Join(root, "internal", "logic", route.Domain, snakeCase(route.HandlerMethod)+".go")
}

func scanHandlers(root string) (map[string]string, error) {
	handlerDir := filepath.Join(root, "internal", "transport", "http", "handler")
	out := map[string]string{}
	return out, scanGoFiles(handlerDir, func(path string, file *ast.File) {
		if strings.HasSuffix(path, "_gen.go") {
			return
		}
		for _, decl := range file.Decls {
			switch item := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range item.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if ok {
						out[typeSpec.Name.Name] = path
					}
				}
			case *ast.FuncDecl:
				if item.Recv == nil {
					out[item.Name.Name] = path
					continue
				}
				recv := receiverTypeName(item.Recv)
				if recv != "" {
					out[recv+"."+item.Name.Name] = path
				}
			}
		}
	})
}

func scanLogics(root string) (map[string]string, error) {
	logicDir := filepath.Join(root, "internal", "logic")
	out := map[string]string{}
	return out, scanGoFiles(logicDir, func(path string, file *ast.File) {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				continue
			}
			receiver := receiverTypeName(fn.Recv)
			// 集中式 Service 用例以方法名对应路由操作，避免生成同义的空 Logic 文件。
			if receiver == "Service" {
				out[logicKey(file.Name.Name, fn.Name.Name)] = path
				continue
			}
			recv := strings.TrimSuffix(receiver, "Logic")
			if recv == "" {
				continue
			}
			out[logicKey(file.Name.Name, recv)] = path
		}
	})
}

func scanGoFiles(dir string, visit func(string, *ast.File)) error {
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	fset := token.NewFileSet()
	return filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		visit(path, file)
		return nil
	})
}

func receiverTypeName(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}
	switch expr := fields.List[0].Type.(type) {
	case *ast.StarExpr:
		if ident, ok := expr.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.Ident:
		return expr.Name
	}
	return ""
}
