package models

import (
	"database/sql/driver"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestUserTableName(t *testing.T) {
	if got := (User{}).TableName(); got != "users" {
		t.Fatalf("TableName = %q, want users", got)
	}
}

func TestRefreshTokenTableName(t *testing.T) {
	if got := (RefreshToken{}).TableName(); got != "refresh_tokens" {
		t.Fatalf("TableName = %q, want refresh_tokens", got)
	}
}

func TestModelConfigTableName(t *testing.T) {
	if got := (ModelConfig{}).TableName(); got != "model_configs" {
		t.Fatalf("TableName = %q, want model_configs", got)
	}
}

func TestUserGormTags(t *testing.T) {
	userType := reflect.TypeOf(User{})
	tests := map[string][]string{
		"ID":             {"type:uuid", "primaryKey"},
		"Username":       {"column:username", "size:64", "uniqueIndex", "not null"},
		"Nickname":       {"column:nickname", "size:64"},
		"Email":          {"column:email", "size:255", "uniqueIndex"},
		"Avatar":         {"column:avatar", "size:512"},
		"PasswordHash":   {"column:password_hash", "size:255", "not null"},
		"BriefingSeenAt": {"column:briefing_seen_at"},
		"CreatedAt":      {"column:created_at", "autoCreateTime"},
		"UpdatedAt":      {"column:updated_at", "autoUpdateTime"},
	}
	for fieldName, wantParts := range tests {
		field, ok := userType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("missing field %s", fieldName)
		}
		tag := field.Tag.Get("gorm")
		for _, want := range wantParts {
			if !strings.Contains(tag, want) {
				t.Fatalf("%s gorm tag = %q, want to contain %q", fieldName, tag, want)
			}
		}
	}
}

func TestModelConfigGormTags(t *testing.T) {
	modelType := reflect.TypeOf(ModelConfig{})
	tests := map[string][]string{
		"ID":              {"column:id", "type:uuid", "primaryKey"},
		"UserID":          {"column:user_id", "type:uuid", "not null", "index"},
		"User":            {"foreignKey:UserID", "references:ID", "OnDelete:CASCADE"},
		"Type":            {"column:type", "size:32", "index", "not null"},
		"Provider":        {"column:provider", "size:32", "not null"},
		"Name":            {"column:name", "size:128", "not null"},
		"ModelName":       {"column:model_name", "size:128", "not null"},
		"APIKeyEncrypted": {"column:api_key_encrypted", "size:512", "not null"},
		"BaseURL":         {"column:base_url", "size:255", "not null"},
		"Capability":      {"column:capability", "type:jsonb"},
		"IsDefault":       {"column:is_default", "default:false"},
		"CreatedAt":       {"column:created_at", "autoCreateTime"},
		"UpdatedAt":       {"column:updated_at", "autoUpdateTime"},
	}
	for fieldName, wantParts := range tests {
		field, ok := modelType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("missing field %s", fieldName)
		}
		tag := field.Tag.Get("gorm")
		for _, want := range wantParts {
			if !strings.Contains(tag, want) {
				t.Fatalf("%s gorm tag = %q, want to contain %q", fieldName, tag, want)
			}
		}
	}
}

func TestKnowledgeBaseGormTags(t *testing.T) {
	// 验证知识库模型包含展示字段和用户隔离所需的 GORM 标签。
	modelType := reflect.TypeOf(KnowledgeBase{})
	tests := map[string][]string{
		"ID":          {"column:id", "type:uuid", "primaryKey"},
		"UserID":      {"column:user_id", "type:uuid", "not null", "index"},
		"User":        {"foreignKey:UserID", "references:ID", "OnDelete:CASCADE"},
		"Name":        {"column:name", "size:128", "not null"},
		"Description": {"column:description", "size:512"},
		"Icon":        {"column:icon", "size:16"},
		"Color":       {"column:color", "size:16", "default:''"},
		"IsDefault":   {"column:is_default", "default:false", "index"},
		"ChatEnabled": {"column:chat_enabled", "default:false"},
		"CreatedAt":   {"column:created_at", "autoCreateTime"},
		"UpdatedAt":   {"column:updated_at", "autoUpdateTime"},
	}
	for fieldName, wantParts := range tests {
		field, ok := modelType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("missing field %s", fieldName)
		}
		tag := field.Tag.Get("gorm")
		for _, want := range wantParts {
			if !strings.Contains(tag, want) {
				t.Fatalf("%s gorm tag = %q, want to contain %q", fieldName, tag, want)
			}
		}
	}
}

func TestStringListScansAndValuesJSON(t *testing.T) {
	var list StringList
	if err := list.Scan([]byte(`["function_call","vision"]`)); err != nil {
		t.Fatalf("Scan error = %v", err)
	}
	if len(list) != 2 || list[0] != "function_call" || list[1] != "vision" {
		t.Fatalf("list = %#v", list)
	}
	value, err := list.Value()
	if err != nil {
		t.Fatalf("Value error = %v", err)
	}
	if _, ok := value.(driver.Value); !ok {
		t.Fatalf("value type = %T, want driver.Value", value)
	}
	if value != `["function_call","vision"]` {
		t.Fatalf("value = %v", value)
	}
}

func TestRefreshTokenGormTags(t *testing.T) {
	tokenType := reflect.TypeOf(RefreshToken{})
	tests := map[string][]string{
		"ID":        {"column:id", "type:uuid", "primaryKey"},
		"UserID":    {"column:user_id", "type:uuid", "not null", "index"},
		"User":      {"foreignKey:UserID", "references:ID", "OnDelete:CASCADE"},
		"TokenHash": {"column:token_hash", "size:128", "uniqueIndex", "not null"},
		"ExpiresAt": {"column:expires_at", "not null", "index"},
		"RevokedAt": {"column:revoked_at", "index"},
		"CreatedAt": {"column:created_at", "autoCreateTime"},
		"UpdatedAt": {"column:updated_at", "autoUpdateTime"},
	}
	for fieldName, wantParts := range tests {
		field, ok := tokenType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("missing field %s", fieldName)
		}
		tag := field.Tag.Get("gorm")
		for _, want := range wantParts {
			if !strings.Contains(tag, want) {
				t.Fatalf("%s gorm tag = %q, want to contain %q", fieldName, tag, want)
			}
		}
	}
}

func TestBeforeCreateAssignsUUIDWhenIDIsNil(t *testing.T) {
	// 验证包含知识库在内的核心模型创建前会自动补齐 UUID。
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "conversation",
			run: func(t *testing.T) {
				row := &Conversation{}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID == uuid.Nil {
					t.Fatal("ID remained nil")
				}
			},
		},
		{
			name: "user",
			run: func(t *testing.T) {
				row := &User{}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID == uuid.Nil {
					t.Fatal("ID remained nil")
				}
			},
		},
		{
			name: "refresh token",
			run: func(t *testing.T) {
				row := &RefreshToken{}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID == uuid.Nil {
					t.Fatal("ID remained nil")
				}
			},
		},
		{
			name: "model config",
			run: func(t *testing.T) {
				row := &ModelConfig{}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID == uuid.Nil {
					t.Fatal("ID remained nil")
				}
			},
		},
		{
			name: "message",
			run: func(t *testing.T) {
				row := &Message{}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID == uuid.Nil {
					t.Fatal("ID remained nil")
				}
			},
		},
		{
			name: "knowledge base",
			run: func(t *testing.T) {
				row := &KnowledgeBase{}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID == uuid.Nil {
					t.Fatal("ID remained nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestBeforeCreatePreservesExistingUUID(t *testing.T) {
	// 验证包含知识库在内的核心模型已有 UUID 时不会被覆盖。
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "conversation",
			run: func(t *testing.T) {
				id := uuid.New()
				row := &Conversation{ID: id}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID != id {
					t.Fatalf("ID = %s, want %s", row.ID, id)
				}
			},
		},
		{
			name: "user",
			run: func(t *testing.T) {
				id := uuid.New()
				row := &User{ID: id}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID != id {
					t.Fatalf("ID = %s, want %s", row.ID, id)
				}
			},
		},
		{
			name: "refresh token",
			run: func(t *testing.T) {
				id := uuid.New()
				row := &RefreshToken{ID: id}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID != id {
					t.Fatalf("ID = %s, want %s", row.ID, id)
				}
			},
		},
		{
			name: "model config",
			run: func(t *testing.T) {
				id := uuid.New()
				row := &ModelConfig{ID: id}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID != id {
					t.Fatalf("ID = %s, want %s", row.ID, id)
				}
			},
		},
		{
			name: "message",
			run: func(t *testing.T) {
				id := uuid.New()
				row := &Message{ID: id}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID != id {
					t.Fatalf("ID = %s, want %s", row.ID, id)
				}
			},
		},
		{
			name: "knowledge base",
			run: func(t *testing.T) {
				id := uuid.New()
				row := &KnowledgeBase{ID: id}
				if err := row.BeforeCreate(nil); err != nil {
					t.Fatalf("BeforeCreate error = %v", err)
				}
				if row.ID != id {
					t.Fatalf("ID = %s, want %s", row.ID, id)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
