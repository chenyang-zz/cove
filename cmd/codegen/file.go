package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func generatedFile(pkg string, imports []string, body string, includeHeader bool) string {
	var b strings.Builder
	if includeHeader {
		b.WriteString(generatedHeader)
		b.WriteString("\n\n")
	}
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	if len(imports) > 0 {
		b.WriteString("import (\n")
		for _, item := range unique(imports) {
			fmt.Fprintf(&b, "\t%s\n", item)
		}
		b.WriteString(")\n\n")
	}
	b.WriteString(body)
	return b.String()
}

func writeGeneratedFile(path, content string, report *Report) error {
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return fmt.Errorf("format generated file %s: %w\n%s", path, err, content)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if data, err := os.ReadFile(path); err == nil {
		if !strings.HasPrefix(string(data), generatedHeader) {
			return fmt.Errorf("refuse to overwrite non-codegen file %s", path)
		}
		if bytes.Equal(data, formatted) {
			report.Add(FileUnchanged, path)
			return nil
		}
		report.Add(FileModified, path)
		return os.WriteFile(path, formatted, 0o644)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	report.Add(FileAdded, path)
	return os.WriteFile(path, formatted, 0o644)
}

func appendGoFile(path string, imports []string, body string, report *Report) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	withImports, err := ensureImports(data, imports)
	if err != nil {
		return err
	}
	next := strings.TrimRight(string(withImports), "\n") + "\n\n" + strings.TrimSpace(body) + "\n"
	formatted, err := format.Source([]byte(next))
	if err != nil {
		return fmt.Errorf("format appended file %s: %w\n%s", path, err, next)
	}
	if bytes.Equal(data, formatted) {
		report.Add(FileUnchanged, path)
		return nil
	}
	report.Add(FileModified, path)
	return os.WriteFile(path, formatted, 0o644)
}

func ensureImports(src []byte, imports []string) ([]byte, error) {
	imports = unique(imports)
	if len(imports) == 0 {
		return src, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	existing := map[string]struct{}{}
	for _, spec := range file.Imports {
		existing[spec.Path.Value] = struct{}{}
	}
	var missing []string
	for _, item := range imports {
		_, pathValue := splitImportSpec(item)
		if pathValue == "" {
			continue
		}
		if _, ok := existing[pathValue]; ok {
			continue
		}
		missing = append(missing, item)
	}
	if len(missing) == 0 {
		return src, nil
	}

	importDecl := firstImportDecl(file)
	if importDecl == nil {
		offset := fset.Position(file.Name.End()).Offset
		block := "\n\nimport (\n"
		for _, item := range missing {
			block += "\t" + item + "\n"
		}
		block += ")\n"
		return insertAt(src, offset, block), nil
	}

	if importDecl.Lparen.IsValid() {
		offset := fset.Position(importDecl.Rparen).Offset
		block := ""
		for _, item := range missing {
			block += "\t" + item + "\n"
		}
		return insertAt(src, offset, block), nil
	}

	if len(importDecl.Specs) == 0 {
		return src, nil
	}
	spec, ok := importDecl.Specs[0].(*ast.ImportSpec)
	if !ok {
		return src, nil
	}
	start := fset.Position(importDecl.Pos()).Offset
	end := fset.Position(importDecl.End()).Offset
	block := "import (\n\t" + astImportSpec(spec) + "\n"
	for _, item := range missing {
		block += "\t" + item + "\n"
	}
	block += ")"
	return replaceRange(src, start, end, block), nil
}

func firstImportDecl(file *ast.File) *ast.GenDecl {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if ok && gen.Tok == token.IMPORT {
			return gen
		}
	}
	return nil
}

func astImportSpec(spec *ast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name + " " + spec.Path.Value
	}
	return spec.Path.Value
}

func splitImportSpec(item string) (name string, pathValue string) {
	item = strings.TrimSpace(item)
	if item == "" {
		return "", ""
	}
	if strings.HasPrefix(item, `"`) {
		return "", item
	}
	parts := strings.Fields(item)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func insertAt(src []byte, offset int, text string) []byte {
	out := make([]byte, 0, len(src)+len(text))
	out = append(out, src[:offset]...)
	out = append(out, text...)
	out = append(out, src[offset:]...)
	return out
}

func replaceRange(src []byte, start, end int, text string) []byte {
	out := make([]byte, 0, len(src)+len(text)-(end-start))
	out = append(out, src[:start]...)
	out = append(out, text...)
	out = append(out, src[end:]...)
	return out
}

func printReport(w io.Writer, report Report, color bool) {
	fmt.Fprintln(w, "codegen:")
	if len(report.Files) == 0 {
		fmt.Fprintln(w, colorize("  = no files changed", ansiGray, color))
		return
	}
	for _, file := range report.Files {
		symbol, colorCode := fileChangeStyle(file.Kind)
		fmt.Fprintf(w, "  %s\n", colorize(fmt.Sprintf("%s %s", symbol, file.Path), colorCode, color))
	}
}

func fileChangeStyle(kind FileChangeKind) (string, string) {
	switch kind {
	case FileAdded:
		return "+", ansiGreen
	case FileModified:
		return "~", ansiYellow
	case FileSkipped, FileUnchanged:
		return "=", ansiGray
	default:
		return "?", ansiGray
	}
}

const (
	ansiReset  = "\x1b[0m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiGray   = "\x1b[90m"
)

func colorize(text, code string, color bool) string {
	if !color {
		return text
	}
	return code + text + ansiReset
}

func unique(items []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func logicKey(domain, method string) string {
	return strings.ToLower(domain) + "." + strings.ToLower(method)
}

func domainFromHandlerType(handlerType string) string {
	name := strings.TrimSuffix(handlerType, "Handler")
	return strings.ToLower(name)
}

func lowerFirst(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func snakeCase(value string) string {
	var b bytes.Buffer
	for i, r := range value {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
