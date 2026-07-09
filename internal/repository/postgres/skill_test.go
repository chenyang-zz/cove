package postgres_test

import (
	"context"
	"testing"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	repositorypostgres "github.com/boxify/api-go/internal/repository/postgres"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// TestSkillRepositoryCRUDIsUserScopedWhenPostgresEnvIsConfigured 验证技能仓储的增删改查会按用户隔离。
func TestSkillRepositoryCRUDIsUserScopedWhenPostgresEnvIsConfigured(t *testing.T) {
	db := newAuthTestDB(t)
	ctx := context.Background()
	userRepo := repositorypostgres.NewUserRepository(db)
	skillRepo := repositorypostgres.NewSkillRepository(db)

	user, err := userRepo.Create(ctx, &models.User{Username: "skill-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create user error = %v", err)
	}
	otherUser, err := userRepo.Create(ctx, &models.User{Username: "skill-other-" + uuid.NewString(), PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("Create other user error = %v", err)
	}
	t.Cleanup(func() {
		db.WithContext(context.Background()).Exec("DELETE FROM skills WHERE user_id IN ?", []uuid.UUID{user.ID, otherUser.ID})
		db.WithContext(context.Background()).Exec("DELETE FROM users WHERE id IN ?", []uuid.UUID{user.ID, otherUser.ID})
	})

	row, err := skillRepo.Create(ctx, user.ID, &models.Skill{
		ID:          uuid.New(),
		Name:        "写作技能",
		Description: "说明",
		Icon:        "🧩",
		Prompt:      "prompt",
		ToolKeys:    models.StringList{"time"},
		Config: models.SkillConfig{
			QuickPrompt: []string{"hello"},
			FewShots:    []models.SkillFewShot{{Input: "input", Output: "output"}},
		},
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("Create skill error = %v", err)
	}
	if _, err := skillRepo.Create(ctx, otherUser.ID, &models.Skill{ID: uuid.New(), Name: "其他技能", Icon: "🧩", Enabled: true}); err != nil {
		t.Fatalf("Create other skill error = %v", err)
	}

	rows, err := skillRepo.List(ctx, user.ID)
	if err != nil {
		t.Fatalf("List error = %v", err)
	}
	if len(rows) != 1 || rows[0].ID != row.ID {
		t.Fatalf("List rows = %+v, want current user skill only", rows)
	}
	if len(rows[0].Config.QuickPrompt) != 1 || rows[0].Config.QuickPrompt[0] != "hello" ||
		len(rows[0].Config.FewShots) != 1 || rows[0].Config.FewShots[0].Output != "output" {
		t.Fatalf("List config = %+v, want structured JSONB config", rows[0].Config)
	}
	if _, err := skillRepo.FindByID(ctx, otherUser.ID, row.ID); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("FindByID as other user error = %v, want not_found", err)
	}

	updated, err := skillRepo.UpdateFields(ctx, user.ID, row.ID, &models.Skill{Name: "新技能", Enabled: false}, repository.NewSkillUpdateFields().Name().Enabled())
	if err != nil {
		t.Fatalf("UpdateFields error = %v", err)
	}
	if updated.Name != "新技能" || updated.Enabled {
		t.Fatalf("updated skill = %+v, want name changed and enabled false", updated)
	}
	if err := skillRepo.Delete(ctx, otherUser.ID, row.ID); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("Delete as other user error = %v, want not_found", err)
	}
	if err := skillRepo.Delete(ctx, user.ID, row.ID); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if _, err := skillRepo.FindByID(ctx, user.ID, row.ID); xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("FindByID after delete error = %v, want not_found", err)
	}
}
