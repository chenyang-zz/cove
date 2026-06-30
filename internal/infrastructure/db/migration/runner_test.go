package migration

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestNewRunnerRejectsBlankDatabaseURL(t *testing.T) {
	_, err := NewRunner(Config{DatabaseURL: "   "})
	if err == nil {
		t.Fatal("NewRunner returned nil error")
	}
}

func TestRunnerUpIncludesKnowledgeBaseModel(t *testing.T) {
	// 验证迁移清单包含知识库模型，避免 knowledge_bases 表或 color 列遗漏。
	source, err := os.ReadFile("runner.go")
	if err != nil {
		t.Fatalf("read runner.go: %v", err)
	}
	if !strings.Contains(string(source), "&models.KnowledgeBase{}") {
		t.Fatal("Runner.Up does not migrate models.KnowledgeBase")
	}
}

func TestRunnerIntegrationWhenPostgresEnvIsConfigured(t *testing.T) {
	// 验证真实 Postgres 迁移会创建知识库表和展示字段。
	url := os.Getenv("POSTGRES_MIGRATION_TEST_URL")
	if url == "" {
		t.Skip("POSTGRES_MIGRATION_TEST_URL is required")
	}

	runner, err := NewRunner(Config{DatabaseURL: url})
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
}
