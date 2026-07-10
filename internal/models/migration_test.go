package models

import "testing"

type migrationTableNamer interface {
	TableName() string
}

// TestMigrationModelsReturnsValidUniqueTables 验证迁移注册表不包含空模型、重复表名，并包含工具配置表。
func TestMigrationModelsReturnsValidUniqueTables(t *testing.T) {
	registered := MigrationModels()
	if len(registered) == 0 {
		t.Fatal("MigrationModels() returned no models")
	}

	tables := make(map[string]struct{}, len(registered))
	for _, model := range registered {
		if model == nil {
			t.Fatal("MigrationModels() contains nil model")
		}
		namer, ok := model.(migrationTableNamer)
		if !ok {
			t.Fatalf("MigrationModels() model %T does not implement TableName", model)
		}
		tableName := namer.TableName()
		if tableName == "" {
			t.Fatalf("MigrationModels() model %T has empty table name", model)
		}
		if _, exists := tables[tableName]; exists {
			t.Fatalf("MigrationModels() contains duplicate table %q", tableName)
		}
		tables[tableName] = struct{}{}
	}
	if _, ok := tables[(ToolConfig{}).TableName()]; !ok {
		t.Fatal("MigrationModels() does not contain ToolConfig")
	}
}

// TestMigrationModelsReturnsIndependentSlices 验证调用方修改返回切片不会污染后续注册表结果。
func TestMigrationModelsReturnsIndependentSlices(t *testing.T) {
	first := MigrationModels()
	first[0] = nil

	second := MigrationModels()
	if second[0] == nil {
		t.Fatal("MigrationModels() returned a shared slice")
	}
}
