package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func GenerateRepository(opts RepositoryOptions) (Report, error) {
	if opts.Root == "" {
		opts.Root = "."
	}
	report := Report{Root: opts.Root, Command: "repository", Mode: generationMode(opts.DryRun, opts.Check)}
	if opts.DryRun && opts.Check {
		return report, fmt.Errorf("--dry-run and --check are mutually exclusive")
	}
	if strings.TrimSpace(opts.Model) == "" {
		return report, fmt.Errorf("codegen repository: -model is required")
	}
	models, err := scanModels(opts.Root)
	if err != nil {
		return report, err
	}
	if opts.Verbose {
		report.AddDiagnostic("info", "repository.models_scanned", fmt.Sprintf("scanned %d GORM models", len(models)), "", "")
	}
	info, ok := models[opts.Model]
	if !ok {
		return report, fmt.Errorf("codegen repository: model %s not found", opts.Model)
	}
	if !info.HasID {
		return report, fmt.Errorf("codegen repository: model %s must have ID uuid.UUID", opts.Model)
	}
	scope, err := repositoryScopeFor(info, opts.Scope)
	if err != nil {
		return report, err
	}
	label := opts.Label
	if strings.TrimSpace(label) == "" {
		label = opts.Model
	}

	if err := generateRepositoryInterface(opts.Root, info, &report); err != nil {
		return report, err
	}
	if err := generatePostgresRepository(opts.Root, info, label, scope, &report); err != nil {
		return report, err
	}
	if err := generateModelHook(opts.Root, info, &report); err != nil {
		return report, err
	}
	return report, nil
}

func generateRepositoryInterface(root string, info ModelInfo, report *Report) error {
	path := filepath.Join(root, "internal", "repository", snakeCase(info.Name)+".go")
	if fileExists(path) {
		report.Add(FileSkipped, path)
		return nil
	}

	modelVar := lowerFirst(info.Name)
	updateFields := repositoryUpdateFields(info)
	imports := []string{
		`"context"`,
		fmt.Sprintf(`"%s/internal/models"`, modulePath),
		`"github.com/google/uuid"`,
	}
	body, err := renderTemplate("repository_interface.gotmpl", map[string]any{
		"Model":        info.Name,
		"ModelVar":     modelVar,
		"UpdateFields": updateFields,
	})
	if err != nil {
		return err
	}
	return writeNewGeneratedFile(path, generatedFile("repository", imports, body, true), report)
}

func repositoryUpdateFields(info ModelInfo) string {
	var b strings.Builder
	for _, field := range updatableModelFields(info) {
		fmt.Fprintf(&b, "func (f *%sUpdateFields) %s() *%sUpdateFields {\n", info.Name, field.Name, info.Name)
		fmt.Fprintf(&b, "\treturn f.add(%q)\n", field.Column)
		b.WriteString("}\n\n")
	}
	return b.String()
}

func generatePostgresRepository(root string, info ModelInfo, label string, scope RepositoryScope, report *Report) error {
	path := filepath.Join(root, "internal", "repository", "postgres", snakeCase(info.Name)+".go")
	if fileExists(path) {
		report.Add(FileSkipped, path)
		return nil
	}

	modelVar := lowerFirst(info.Name)
	orderColumn := repositoryOrderColumn(info)
	tableName := snakeCase(info.Name) + "s"
	imports := []string{
		`"context"`,
		`"errors"`,
		fmt.Sprintf(`"%s/internal/models"`, modulePath),
		fmt.Sprintf(`"%s/internal/repository"`, modulePath),
		fmt.Sprintf(`"%s/internal/xerr"`, modulePath),
		`"github.com/google/uuid"`,
		`"gorm.io/gorm"`,
	}
	body, err := renderTemplate("repository_postgres.gotmpl", map[string]any{
		"Model":             info.Name,
		"ModelVar":          modelVar,
		"OrderColumn":       orderColumn,
		"Label":             label,
		"CreateScope":       createScopeSnippet(scope, info, modelVar, label),
		"ListScope":         listScopeSnippet(scope, tableName),
		"FindScope":         findScopeSnippet(scope, tableName, modelVar),
		"UpdateScope":       updateScopeSnippet(scope, tableName, modelVar, modelVar+".ID"),
		"UpdateFieldsScope": updateScopeSnippet(scope, tableName, modelVar, modelVar+"ID"),
		"DeleteScope":       updateScopeSnippet(scope, tableName, modelVar, modelVar+"ID"),
	})
	if err != nil {
		return err
	}
	return writeNewGeneratedFile(path, generatedFile("postgres", imports, body, true), report)
}

func repositoryOrderColumn(info ModelInfo) string {
	if info.HasUpdatedAt {
		return "updated_at"
	}
	if info.HasCreatedAt {
		return "created_at"
	}
	return "id"
}

func repositoryScopeFor(info ModelInfo, raw string) (RepositoryScope, error) {
	raw = strings.TrimSpace(raw)
	if info.HasUserID && raw == "" {
		return RepositoryScope{Kind: "direct", UserColumn: "user_id"}, nil
	}
	if raw == "" {
		return RepositoryScope{}, fmt.Errorf("codegen repository: model %s must have UserID uuid.UUID or provide -scope local_column:table.column:user_column, for example -scope conversation_id:conversations.id:user_id", info.Name)
	}
	scope, err := parseRepositoryScope(raw)
	if err != nil {
		return RepositoryScope{}, err
	}
	if !modelHasColumn(info, scope.LocalColumn) {
		return RepositoryScope{}, fmt.Errorf("codegen repository: model %s does not have scope local column %s", info.Name, scope.LocalColumn)
	}
	return scope, nil
}

func parseRepositoryScope(raw string) (RepositoryScope, error) {
	parts := strings.Split(raw, ":")
	if len(parts) != 3 {
		return RepositoryScope{}, fmt.Errorf("codegen repository: invalid scope %q, want local_column:table.column:user_column", raw)
	}
	join := strings.Split(parts[1], ".")
	if len(join) != 2 {
		return RepositoryScope{}, fmt.Errorf("codegen repository: invalid scope %q, want local_column:table.column:user_column", raw)
	}
	scope := RepositoryScope{
		Kind:        "join",
		LocalColumn: strings.TrimSpace(parts[0]),
		JoinTable:   strings.TrimSpace(join[0]),
		JoinColumn:  strings.TrimSpace(join[1]),
		UserColumn:  strings.TrimSpace(parts[2]),
	}
	if scope.LocalColumn == "" || scope.JoinTable == "" || scope.JoinColumn == "" || scope.UserColumn == "" {
		return RepositoryScope{}, fmt.Errorf("codegen repository: invalid scope %q, want local_column:table.column:user_column", raw)
	}
	return scope, nil
}

func modelHasColumn(info ModelInfo, column string) bool {
	for _, field := range info.Fields {
		if field.Column == column {
			return true
		}
	}
	return false
}

func createScopeSnippet(scope RepositoryScope, info ModelInfo, modelVar string, label string) string {
	if scope.Kind == "direct" {
		return fmt.Sprintf("\t%s.UserID = userID", modelVar)
	}
	fieldName := fieldNameByColumn(info, scope.LocalColumn)
	return fmt.Sprintf(`	var scopeCount int64
	if err := r.db.WithContext(ctx).
		Table(%q).
		Where("%s = ? AND %s = ?", %s.%s, userID).
		Count(&scopeCount).Error; err != nil {
		return nil, xerr.Wrapf(err, "校验数据归属失败")
	}
	if scopeCount == 0 {
		return nil, xerr.NotFound("%s不存在")
	}`, scope.JoinTable, scope.JoinColumn, scope.UserColumn, modelVar, fieldName, label)
}

func listScopeSnippet(scope RepositoryScope, tableName string) string {
	if scope.Kind == "direct" {
		return "\n\t\tWhere(\"user_id = ?\", userID)."
	}
	return fmt.Sprintf(`
		Joins("JOIN %s ON %s.%s = %s.%s").
		Where("%s.%s = ?", userID).`, scope.JoinTable, tableName, scope.LocalColumn, scope.JoinTable, scope.JoinColumn, scope.JoinTable, scope.UserColumn)
}

func findScopeSnippet(scope RepositoryScope, tableName string, modelVar string) string {
	if scope.Kind == "direct" {
		return fmt.Sprintf("\n\t\tWhere(\"id = ? AND user_id = ?\", %sID, userID).", modelVar)
	}
	return fmt.Sprintf(`
		Joins("JOIN %s ON %s.%s = %s.%s").
		Where("%s.id = ?", %sID).
		Where("%s.%s = ?", userID).`, scope.JoinTable, tableName, scope.LocalColumn, scope.JoinTable, scope.JoinColumn, tableName, modelVar, scope.JoinTable, scope.UserColumn)
}

func updateScopeSnippet(scope RepositoryScope, tableName string, modelVar string, idExpr string) string {
	if scope.Kind == "direct" {
		return fmt.Sprintf("\t\tWhere(\"id = ? AND user_id = ?\", %s, userID).", idExpr)
	}
	return fmt.Sprintf(`		Where("id = ?", %s).
		Where("EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s AND %s.%s = ?)", userID).`, idExpr, scope.JoinTable, scope.JoinTable, scope.JoinColumn, tableName, scope.LocalColumn, scope.JoinTable, scope.UserColumn)
}

func fieldNameByColumn(info ModelInfo, column string) string {
	for _, field := range info.Fields {
		if field.Column == column {
			return field.Name
		}
	}
	return ""
}

func updatableModelFields(info ModelInfo) []ModelField {
	skip := map[string]struct{}{
		"ID":        {},
		"UserID":    {},
		"User":      {},
		"CreatedAt": {},
		"UpdatedAt": {},
	}
	var out []ModelField
	for _, field := range info.Fields {
		if field.Column == "" {
			continue
		}
		if _, ok := skip[field.Name]; ok {
			continue
		}
		out = append(out, field)
	}
	return out
}

func generateModelHook(root string, info ModelInfo, report *Report) error {
	path := filepath.Join(root, "internal", "models", "hooks.go")
	hook, err := modelBeforeCreateHook(info)
	if err != nil {
		return err
	}
	if !fileExists(path) {
		imports := []string{
			`"github.com/google/uuid"`,
			`"gorm.io/gorm"`,
		}
		body, err := renderTemplate("model_hooks.gotmpl", map[string]any{
			"IncludeEnsureUUID": true,
			"Hook":              hook,
		})
		if err != nil {
			return err
		}
		return writeNewGeneratedFile(path, generatedFile("models", imports, body, false), report)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if hasBeforeCreateHook(data, info.Name) {
		report.Add(FileSkipped, path)
		return nil
	}

	next := string(data)
	if !strings.Contains(next, "func ensureUUID(id *uuid.UUID)") {
		ensureUUID, err := renderTemplate("model_hooks.gotmpl", map[string]any{
			"IncludeEnsureUUID": true,
			"Hook":              "",
		})
		if err != nil {
			return err
		}
		next = strings.TrimRight(next, "\n") + "\n\n" + ensureUUID
	}
	next = strings.TrimRight(next, "\n") + "\n\n" + hook
	withImports, err := ensureImports([]byte(next), []string{
		`"github.com/google/uuid"`,
		`"gorm.io/gorm"`,
	})
	if err != nil {
		return err
	}
	formatted, err := format.Source(withImports)
	if err != nil {
		return fmt.Errorf("format model hooks file %s: %w\n%s", path, err, string(withImports))
	}
	if string(data) == string(formatted) {
		report.Add(FileUnchanged, path)
		return nil
	}
	if report.IsPreview() {
		report.Add(FileWouldModify, path)
		return nil
	}
	report.Add(FileModified, path)
	return os.WriteFile(path, formatted, 0o644)
}

func modelBeforeCreateHook(info ModelInfo) (string, error) {
	receiver := modelHookReceiver(info.Name)
	return renderTemplate("model_before_create.gotmpl", map[string]any{
		"Receiver": receiver,
		"Model":    info.Name,
	})
}

func modelHookReceiver(modelName string) string {
	if modelName == "" {
		return "m"
	}
	runes := []rune(modelName)
	return strings.ToLower(string(runes[0]))
}

func hasBeforeCreateHook(src []byte, modelName string) bool {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return strings.Contains(string(src), "*"+modelName+") BeforeCreate")
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "BeforeCreate" || fn.Recv == nil || len(fn.Recv.List) == 0 {
			continue
		}
		if receiverTypeName(fn.Recv) == modelName {
			return true
		}
	}
	return false
}

func writeNewGeneratedFile(path, content string, report *Report) error {
	if fileExists(path) {
		report.Add(FileSkipped, path)
		return nil
	}
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return fmt.Errorf("format generated file %s: %w\n%s", path, err, content)
	}
	if report.IsPreview() {
		report.Add(FileWouldAdd, path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	report.Add(FileAdded, path)
	return os.WriteFile(path, formatted, 0o644)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func sortedModelFieldNames(fields []ModelField) []string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}
	sort.Strings(names)
	return names
}
