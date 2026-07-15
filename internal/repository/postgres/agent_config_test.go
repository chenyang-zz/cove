package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	repositorypostgres "github.com/boxify/api-go/internal/repository/postgres"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// TestAgentConfigRepositorySupportsMultipleUserScopedRows 验证同一用户可保存多条配置，并且查询、更新和删除均按用户及配置 ID 隔离。
func TestAgentConfigRepositorySupportsMultipleUserScopedRows(t *testing.T) {
	db := newAuthTestDB(t)
	ctx := context.Background()
	userRepo := repositorypostgres.NewUserRepository(db)
	configRepo := repositorypostgres.NewAgentConfigRepository(db)

	owner, err := userRepo.Create(ctx, &models.User{Username: "agent-config-owner-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create owner error = %v", err)
	}
	other, err := userRepo.Create(ctx, &models.User{Username: "agent-config-other-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create other error = %v", err)
	}
	t.Cleanup(func() {
		db.WithContext(context.Background()).Exec("DELETE FROM agent_configs WHERE user_id IN ?", []uuid.UUID{owner.ID, other.ID})
		db.WithContext(context.Background()).Exec("DELETE FROM users WHERE id IN ?", []uuid.UUID{owner.ID, other.ID})
	})

	first, err := configRepo.Create(ctx, owner.ID, &models.AgentConfig{SystemPrompt: "first", Temperature: 0.7})
	if err != nil {
		t.Fatalf("Create first config error = %v", err)
	}
	second, err := configRepo.Create(ctx, owner.ID, &models.AgentConfig{SystemPrompt: "second", Temperature: 0.8})
	if err != nil {
		t.Fatalf("Create second config error = %v", err)
	}
	foreign, err := configRepo.Create(ctx, other.ID, &models.AgentConfig{SystemPrompt: "foreign", Temperature: 0.9})
	if err != nil {
		t.Fatalf("Create foreign config error = %v", err)
	}
	if first.Name != "默认配置" || second.Name != "智能体配置 2" || foreign.Name != "默认配置" {
		t.Fatalf("generated config names = first:%q second:%q foreign:%q, want user-scoped defaults", first.Name, second.Name, foreign.Name)
	}
	if _, err := configRepo.Create(ctx, owner.ID, &models.AgentConfig{Name: first.Name, Temperature: 0.7}); xerr.From(err).Kind != xerr.KindConflict {
		t.Fatalf("Create duplicate name error = %v, want conflict", err)
	}

	older := time.Now().Add(-time.Hour)
	newer := time.Now()
	if err := db.Model(&models.AgentConfig{}).Where("id = ?", first.ID).UpdateColumn("updated_at", older).Error; err != nil {
		t.Fatalf("set first updated_at error = %v", err)
	}
	if err := db.Model(&models.AgentConfig{}).Where("id = ?", second.ID).UpdateColumn("updated_at", newer).Error; err != nil {
		t.Fatalf("set second updated_at error = %v", err)
	}

	rows, err := configRepo.List(ctx, owner.ID)
	if err != nil {
		t.Fatalf("List owner configs error = %v", err)
	}
	if len(rows) != 2 || rows[0].ID != second.ID || rows[1].ID != first.ID {
		t.Fatalf("List owner configs IDs = %v, want [%s %s]", []uuid.UUID{rows[0].ID, rows[1].ID}, second.ID, first.ID)
	}
	if _, err := configRepo.FindByID(ctx, other.ID, first.ID); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("FindByID as another user error = %v, want not found", err)
	}

	patch := *first
	patch.SystemPrompt = "updated"
	updated, err := configRepo.UpdateFields(ctx, owner.ID, first.ID, &patch, repository.NewAgentConfigUpdateFields().SystemPrompt())
	if err != nil {
		t.Fatalf("UpdateFields owner config error = %v", err)
	}
	if updated.SystemPrompt != "updated" || updated.Temperature != first.Temperature {
		t.Fatalf("UpdateFields result = %#v, want updated prompt and unchanged temperature", updated)
	}
	duplicateNamePatch := *first
	duplicateNamePatch.Name = second.Name
	if _, err := configRepo.UpdateFields(ctx, owner.ID, first.ID, &duplicateNamePatch, repository.NewAgentConfigUpdateFields().Name()); xerr.From(err).Kind != xerr.KindConflict {
		t.Fatalf("UpdateFields duplicate name error = %v, want conflict", err)
	}
	renamedPatch := *first
	renamedPatch.Name = "工作助手"
	renamed, err := configRepo.UpdateFields(ctx, owner.ID, first.ID, &renamedPatch, repository.NewAgentConfigUpdateFields().Name())
	if err != nil || renamed.Name != "工作助手" {
		t.Fatalf("UpdateFields name result = %#v error=%v, want 工作助手", renamed, err)
	}
	if _, err := configRepo.UpdateFields(ctx, other.ID, first.ID, &patch, repository.NewAgentConfigUpdateFields().SystemPrompt()); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("UpdateFields as another user error = %v, want not found", err)
	}

	if err := configRepo.Delete(ctx, other.ID, first.ID); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("Delete as another user error = %v, want not found", err)
	}
	if err := configRepo.Delete(ctx, owner.ID, first.ID); err != nil {
		t.Fatalf("Delete owner config error = %v", err)
	}
	if _, err := configRepo.FindByID(ctx, owner.ID, first.ID); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("FindByID deleted config error = %v, want not found", err)
	}
	if found, err := configRepo.FindByID(ctx, other.ID, foreign.ID); err != nil || found.ID != foreign.ID {
		t.Fatalf("FindByID foreign owner result = %#v, error=%v, want own row", found, err)
	}
}

// TestAgentConfigRepositoryMaintainsDefaultInvariant 验证首条配置自动默认、切换默认、旧数据修复和删除后提升均保持唯一默认项。
func TestAgentConfigRepositoryMaintainsDefaultInvariant(t *testing.T) {
	db := newAuthTestDB(t)
	ctx := context.Background()
	userRepo := repositorypostgres.NewUserRepository(db)
	configRepo := repositorypostgres.NewAgentConfigRepository(db)

	owner, err := userRepo.Create(ctx, &models.User{Username: "agent-default-owner-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create default owner error = %v", err)
	}
	other, err := userRepo.Create(ctx, &models.User{Username: "agent-default-other-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create default other error = %v", err)
	}
	t.Cleanup(func() {
		db.WithContext(context.Background()).Exec("DELETE FROM agent_configs WHERE user_id IN ?", []uuid.UUID{owner.ID, other.ID})
		db.WithContext(context.Background()).Exec("DELETE FROM users WHERE id IN ?", []uuid.UUID{owner.ID, other.ID})
	})

	first, err := configRepo.Create(ctx, owner.ID, &models.AgentConfig{SystemPrompt: "first", Temperature: 0.7})
	if err != nil {
		t.Fatalf("Create first default config error = %v", err)
	}
	second, err := configRepo.Create(ctx, owner.ID, &models.AgentConfig{SystemPrompt: "second", Temperature: 0.8})
	if err != nil {
		t.Fatalf("Create second default config error = %v", err)
	}
	if !first.IsDefault || second.IsDefault {
		t.Fatalf("Create default flags = first:%v second:%v, want true false", first.IsDefault, second.IsDefault)
	}

	selected, err := configRepo.SetDefault(ctx, owner.ID, second.ID)
	if err != nil || !selected.IsDefault {
		t.Fatalf("SetDefault(second) result=%#v error=%v, want default second", selected, err)
	}
	if _, err := configRepo.SetDefault(ctx, other.ID, second.ID); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("SetDefault(other user) error=%v, want not found", err)
	}
	rows, err := configRepo.List(ctx, owner.ID)
	if err != nil {
		t.Fatalf("List after SetDefault error = %v", err)
	}
	defaults := 0
	for _, row := range rows {
		if row.IsDefault {
			defaults++
		}
	}
	if defaults != 1 {
		t.Fatalf("default count after SetDefault = %d, want 1", defaults)
	}

	if err := db.Model(&models.AgentConfig{}).Where("user_id = ?", owner.ID).Update("is_default", false).Error; err != nil {
		t.Fatalf("clear defaults for legacy repair error = %v", err)
	}
	repaired, err := configRepo.FindDefault(ctx, owner.ID)
	if err != nil || !repaired.IsDefault {
		t.Fatalf("FindDefault legacy repair result=%#v error=%v, want repaired default", repaired, err)
	}
	if err := configRepo.Delete(ctx, owner.ID, repaired.ID); err != nil {
		t.Fatalf("Delete repaired default error = %v", err)
	}
	promoted, err := configRepo.FindDefault(ctx, owner.ID)
	if err != nil || !promoted.IsDefault || promoted.ID == repaired.ID {
		t.Fatalf("FindDefault after delete result=%#v error=%v, want promoted remaining config", promoted, err)
	}
}
