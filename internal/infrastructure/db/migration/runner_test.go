package migration

import (
	"context"
	"os"
	"testing"

	"github.com/boxify/api-go/internal/models"
)

// TestNewRunnerRejectsBlankDatabaseURL 验证空数据库连接地址会在建立连接前被拒绝。
func TestNewRunnerRejectsBlankDatabaseURL(t *testing.T) {
	_, err := NewRunner(Config{DatabaseURL: "   "})
	if err == nil {
		t.Fatal("NewRunner returned nil error")
	}
}

// TestNewRunnerRejectsEmptyMigrationModels 验证空模型列表会在建立数据库连接前被拒绝。
func TestNewRunnerRejectsEmptyMigrationModels(t *testing.T) {
	_, err := NewRunner(Config{DatabaseURL: "postgres://localhost/test"})
	if err == nil {
		t.Fatal("NewRunner returned nil error")
	}
}

// TestNewRunnerCopiesMigrationModels 验证 runner 保存独立的模型切片，避免调用方后续修改影响迁移。
func TestNewRunnerCopiesMigrationModels(t *testing.T) {
	registered := []any{&models.User{}}
	runner := newRunner(nil, registered...)
	registered[0] = nil

	if runner.models[0] == nil {
		t.Fatal("newRunner stored caller-owned migration models slice")
	}
}

// TestRunnerIntegrationWhenPostgresEnvIsConfigured 验证真实 Postgres 迁移会创建注册模型对应的表和字段。
func TestRunnerIntegrationWhenPostgresEnvIsConfigured(t *testing.T) {
	url := os.Getenv("POSTGRES_MIGRATION_TEST_URL")
	if url == "" {
		t.Skip("POSTGRES_MIGRATION_TEST_URL is required")
	}

	runner, err := NewRunner(Config{DatabaseURL: url}, models.MigrationModels()...)
	if err != nil {
		t.Fatalf("NewRunner error = %v", err)
	}
	defer func() {
		if err := runner.Close(); err != nil {
			t.Fatalf("Close error = %v", err)
		}
	}()

	if err := runner.Up(context.Background()); err != nil {
		t.Fatalf("Up error = %v", err)
	}

	db, err := runner.DB()
	if err != nil {
		t.Fatalf("DB error = %v", err)
	}
	for _, column := range []string{"id", "username", "nickname", "email", "avatar", "password_hash", "briefing_seen_at", "created_at", "updated_at"} {
		var exists bool
		err := db.QueryRowContext(context.Background(), `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'users' AND column_name = $1
			)`, column).Scan(&exists)
		if err != nil {
			t.Fatalf("query column %s: %v", column, err)
		}
		if !exists {
			t.Fatalf("column %s does not exist", column)
		}
	}
	for _, column := range []string{"username", "email"} {
		var exists bool
		err := db.QueryRowContext(context.Background(), `
			SELECT EXISTS (
				SELECT 1
				FROM pg_indexes
				WHERE tablename = 'users'
				  AND indexdef ILIKE '%UNIQUE%'
				  AND indexdef ILIKE '%' || $1 || '%'
			)`, column).Scan(&exists)
		if err != nil {
			t.Fatalf("query unique index %s: %v", column, err)
		}
		if !exists {
			t.Fatalf("unique index for %s does not exist", column)
		}
	}
	for _, column := range []string{"id", "user_id", "token_hash", "expires_at", "revoked_at", "created_at", "updated_at"} {
		var exists bool
		err := db.QueryRowContext(context.Background(), `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'refresh_tokens' AND column_name = $1
			)`, column).Scan(&exists)
		if err != nil {
			t.Fatalf("query refresh token column %s: %v", column, err)
		}
		if !exists {
			t.Fatalf("refresh token column %s does not exist", column)
		}
	}
	var tokenHashUnique bool
	err = db.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM pg_indexes
			WHERE tablename = 'refresh_tokens'
			  AND indexdef ILIKE '%UNIQUE%'
			  AND indexdef ILIKE '%token_hash%'
		)`).Scan(&tokenHashUnique)
	if err != nil {
		t.Fatalf("query token_hash unique index: %v", err)
	}
	if !tokenHashUnique {
		t.Fatal("unique index for token_hash does not exist")
	}
	for _, column := range []string{"id", "user_id", "name", "description", "icon", "color", "is_default", "chat_enabled", "created_at", "updated_at"} {
		var exists bool
		err := db.QueryRowContext(context.Background(), `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'knowledge_bases' AND column_name = $1
			)`, column).Scan(&exists)
		if err != nil {
			t.Fatalf("query knowledge base column %s: %v", column, err)
		}
		if !exists {
			t.Fatalf("knowledge base column %s does not exist", column)
		}
	}
	var toolConfigsExists bool
	err = db.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'tool_configs'
		)`).Scan(&toolConfigsExists)
	if err != nil {
		t.Fatalf("query tool_configs table: %v", err)
	}
	if !toolConfigsExists {
		t.Fatal("tool_configs table does not exist")
	}
}
