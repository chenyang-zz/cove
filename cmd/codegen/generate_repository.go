package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func GenerateRepository(opts RepositoryOptions) (Report, error) {
	if opts.Root == "" {
		opts.Root = "."
	}
	report := Report{Root: opts.Root}
	if strings.TrimSpace(opts.Model) == "" {
		return report, fmt.Errorf("codegen repository: -model is required")
	}
	models, err := scanModels(opts.Root)
	if err != nil {
		return report, err
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
	body := fmt.Sprintf(`type %[1]sRepository interface {
	Create(ctx context.Context, userID uuid.UUID, %[2]s *models.%[1]s) (*models.%[1]s, error)
	List(ctx context.Context, userID uuid.UUID) ([]*models.%[1]s, error)
	FindByID(ctx context.Context, userID uuid.UUID, %[2]sID uuid.UUID) (*models.%[1]s, error)
	Update(ctx context.Context, userID uuid.UUID, %[2]s *models.%[1]s) (*models.%[1]s, error)
	UpdateFields(ctx context.Context, userID uuid.UUID, %[2]sID uuid.UUID, %[2]s *models.%[1]s, fields *%[1]sUpdateFields) (*models.%[1]s, error)
	Delete(ctx context.Context, userID uuid.UUID, %[2]sID uuid.UUID) error
}

type %[1]sUpdateFields struct {
	columns []string
	seen    map[string]struct{}
}

func New%[1]sUpdateFields() *%[1]sUpdateFields {
	return &%[1]sUpdateFields{
		seen: map[string]struct{}{},
	}
}

%[3]s
func (f *%[1]sUpdateFields) Columns() []string {
	if f == nil || len(f.columns) == 0 {
		return nil
	}
	out := make([]string, len(f.columns))
	copy(out, f.columns)
	return out
}

func (f *%[1]sUpdateFields) add(column string) *%[1]sUpdateFields {
	if f == nil {
		f = New%[1]sUpdateFields()
	}
	if f.seen == nil {
		f.seen = map[string]struct{}{}
	}
	if _, ok := f.seen[column]; ok {
		return f
	}
	f.seen[column] = struct{}{}
	f.columns = append(f.columns, column)
	return f
}
`, info.Name, modelVar, updateFields)
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
	body := fmt.Sprintf(`type %[1]sRepository struct {
	db *gorm.DB
}

func New%[1]sRepository(db *gorm.DB) repository.%[1]sRepository {
	return &%[1]sRepository{db: db}
}

func (r *%[1]sRepository) Create(ctx context.Context, userID uuid.UUID, %[2]s *models.%[1]s) (*models.%[1]s, error) {
%[5]s
	if err := r.db.WithContext(ctx).Create(%[2]s).Error; err != nil {
		return nil, xerr.Wrapf(err, "创建%[4]s失败")
	}
	return %[2]s, nil
}

func (r *%[1]sRepository) List(ctx context.Context, userID uuid.UUID) ([]*models.%[1]s, error) {
	var rows []*models.%[1]s

	err := r.db.WithContext(ctx).%[6]s
		Order("%[3]s DESC").
		Find(&rows).Error
	if err != nil {
		return nil, xerr.Wrapf(err, "查询%[4]s列表失败")
	}

	return rows, nil
}

func (r *%[1]sRepository) FindByID(ctx context.Context, userID uuid.UUID, %[2]sID uuid.UUID) (*models.%[1]s, error) {
	%[2]s := &models.%[1]s{}
	err := r.db.WithContext(ctx).%[7]s
		First(%[2]s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NotFound("%[4]s不存在")
	}
	if err != nil {
		return nil, xerr.Wrapf(err, "查询%[4]s失败")
	}
	return %[2]s, nil
}

func (r *%[1]sRepository) Update(ctx context.Context, userID uuid.UUID, %[2]s *models.%[1]s) (*models.%[1]s, error) {
	result := r.db.WithContext(ctx).
		Model(&models.%[1]s{}).
%[8]s
		Omit("id", "user_id", "user", "created_at", "updated_at").
		Updates(%[2]s)
	if result.Error != nil {
		return nil, xerr.Wrapf(result.Error, "更新%[4]s失败")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NotFound("%[4]s不存在")
	}
	return r.FindByID(ctx, userID, %[2]s.ID)
}

func (r *%[1]sRepository) UpdateFields(ctx context.Context, userID uuid.UUID, %[2]sID uuid.UUID, %[2]s *models.%[1]s, fields *repository.%[1]sUpdateFields) (*models.%[1]s, error) {
	columns := fields.Columns()
	if len(columns) == 0 {
		return nil, xerr.BadRequest("更新字段不能为空")
	}
	result := r.db.WithContext(ctx).
		Model(&models.%[1]s{}).
%[9]s
		Select(columns).
		Updates(%[2]s)
	if result.Error != nil {
		return nil, xerr.Wrapf(result.Error, "更新%[4]s失败")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NotFound("%[4]s不存在")
	}
	return r.FindByID(ctx, userID, %[2]sID)
}

func (r *%[1]sRepository) Delete(ctx context.Context, userID uuid.UUID, %[2]sID uuid.UUID) error {
	result := r.db.WithContext(ctx).
%[10]s
		Delete(&models.%[1]s{})
	if result.Error != nil {
		return xerr.Wrapf(result.Error, "删除%[4]s失败")
	}
	if result.RowsAffected == 0 {
		return xerr.NotFound("%[4]s不存在")
	}
	return nil
}
`, info.Name, modelVar, orderColumn, label,
		createScopeSnippet(scope, info, modelVar, label),
		listScopeSnippet(scope, tableName),
		findScopeSnippet(scope, tableName, modelVar),
		updateScopeSnippet(scope, tableName, modelVar, modelVar+".ID"),
		updateScopeSnippet(scope, tableName, modelVar, modelVar+"ID"),
		updateScopeSnippet(scope, tableName, modelVar, modelVar+"ID"),
	)
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
		return RepositoryScope{}, fmt.Errorf("codegen repository: model %s must have UserID uuid.UUID or provide -scope local_column:table.column:user_column", info.Name)
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

func writeNewGeneratedFile(path, content string, report *Report) error {
	if fileExists(path) {
		report.Add(FileSkipped, path)
		return nil
	}
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return fmt.Errorf("format generated file %s: %w\n%s", path, err, content)
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
