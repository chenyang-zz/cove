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
	if !info.HasUserID {
		return report, fmt.Errorf("codegen repository: model %s must have UserID uuid.UUID", opts.Model)
	}
	label := opts.Label
	if strings.TrimSpace(label) == "" {
		label = opts.Model
	}

	if err := generateRepositoryInterface(opts.Root, info, &report); err != nil {
		return report, err
	}
	if err := generatePostgresRepository(opts.Root, info, label, &report); err != nil {
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

func generatePostgresRepository(root string, info ModelInfo, label string, report *Report) error {
	path := filepath.Join(root, "internal", "repository", "postgres", snakeCase(info.Name)+".go")
	if fileExists(path) {
		report.Add(FileSkipped, path)
		return nil
	}

	modelVar := lowerFirst(info.Name)
	orderColumn := repositoryOrderColumn(info)
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
	%[2]s.UserID = userID
	if err := r.db.WithContext(ctx).Create(%[2]s).Error; err != nil {
		return nil, xerr.Wrapf(err, "创建%[4]s失败")
	}
	return %[2]s, nil
}

func (r *%[1]sRepository) List(ctx context.Context, userID uuid.UUID) ([]*models.%[1]s, error) {
	var rows []*models.%[1]s

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("%[3]s DESC").
		Find(&rows).Error
	if err != nil {
		return nil, xerr.Wrapf(err, "查询%[4]s列表失败")
	}

	return rows, nil
}

func (r *%[1]sRepository) FindByID(ctx context.Context, userID uuid.UUID, %[2]sID uuid.UUID) (*models.%[1]s, error) {
	%[2]s := &models.%[1]s{}
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", %[2]sID, userID).
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
		Where("id = ? AND user_id = ?", %[2]s.ID, userID).
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
		Where("id = ? AND user_id = ?", %[2]sID, userID).
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
		Where("id = ? AND user_id = ?", %[2]sID, userID).
		Delete(&models.%[1]s{})
	if result.Error != nil {
		return xerr.Wrapf(result.Error, "删除%[4]s失败")
	}
	if result.RowsAffected == 0 {
		return xerr.NotFound("%[4]s不存在")
	}
	return nil
}
`, info.Name, modelVar, orderColumn, label)
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
