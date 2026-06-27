package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func scanModels(root string) (map[string]ModelInfo, error) {
	modelsDir := filepath.Join(root, "internal", "models")
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	out := map[string]ModelInfo{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(modelsDir, entry.Name())
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, err
		}
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
				out[typeSpec.Name.Name] = modelInfoFromStruct(fset, typeSpec.Name.Name, structType)
			}
		}
	}
	return out, nil
}

func modelInfoFromStruct(fset *token.FileSet, name string, structType *ast.StructType) ModelInfo {
	info := ModelInfo{Name: name}
	if structType == nil || structType.Fields == nil {
		return info
	}
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		column := ""
		if field.Tag != nil {
			column = gormColumn(field.Tag.Value)
		}
		for _, fieldName := range field.Names {
			item := ModelField{
				Name:   fieldName.Name,
				Type:   exprString(fset, field.Type),
				Column: column,
			}
			info.Fields = append(info.Fields, item)
			switch fieldName.Name {
			case "ID":
				info.HasID = item.Type == "uuid.UUID"
			case "UserID":
				info.HasUserID = item.Type == "uuid.UUID"
			case "CreatedAt":
				info.HasCreatedAt = true
			case "UpdatedAt":
				info.HasUpdatedAt = true
			}
		}
	}
	return info
}

func gormColumn(raw string) string {
	tag := tagValue(raw, "gorm")
	for _, part := range strings.Split(tag, ";") {
		if strings.HasPrefix(part, "column:") {
			return strings.TrimPrefix(part, "column:")
		}
	}
	return ""
}

func exprString(fset *token.FileSet, expr ast.Expr) string {
	var b bytes.Buffer
	if err := format.Node(&b, fset, expr); err != nil {
		return ""
	}
	return b.String()
}
