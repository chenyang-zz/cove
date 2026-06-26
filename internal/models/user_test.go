package models

import (
	"reflect"
	"strings"
	"testing"
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

func TestRefreshTokenGormTags(t *testing.T) {
	tokenType := reflect.TypeOf(RefreshToken{})
	tests := map[string][]string{
		"ID":        {"column:id", "type:uuid", "primaryKey"},
		"UserID":    {"column:user_id", "type:uuid", "not null", "index"},
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
