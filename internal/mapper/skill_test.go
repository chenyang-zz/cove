package mapper_test

import (
	"testing"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
)

// TestSkillConfigFromRequestMapsTypedConfig 验证 mapper 会把请求 DTO 转换为模型层技能配置。
func TestSkillConfigFromRequestMapsTypedConfig(t *testing.T) {
	got := mapper.SkillConfigFromRequest(&request.SkillConfig{
		QuickPrompt: []string{"问题"},
		FewShots:    []request.FewShot{{Input: "输入", Output: "输出"}},
	})
	if len(got.QuickPrompt) != 1 || got.QuickPrompt[0] != "问题" ||
		len(got.FewShots) != 1 || got.FewShots[0].Input != "输入" || got.FewShots[0].Output != "输出" {
		t.Fatalf("SkillConfigFromRequest = %+v, want typed model config", got)
	}
}

// TestSkillToResponseReturnsTypedConfig 验证技能响应直接映射模型层结构化配置。
func TestSkillToResponseReturnsTypedConfig(t *testing.T) {
	row := &models.Skill{
		ID:   uuid.New(),
		Name: "技能",
		Config: models.SkillConfig{
			QuickPrompt: []string{"问题"},
			FewShots:    []models.SkillFewShot{{Input: "输入", Output: "输出"}},
		},
	}
	got := mapper.SkillToResponse(row)
	var requestConfig *request.SkillConfig = got.Config
	if got.Config == nil || len(got.Config.QuickPrompt) != 1 || got.Config.QuickPrompt[0] != "问题" ||
		len(got.Config.FewShots) != 1 || got.Config.FewShots[0].Output != "输出" {
		t.Fatalf("SkillToResponse Config = %+v, want typed response config", got.Config)
	}
	if requestConfig.FewShots[0].Input != "输入" {
		t.Fatalf("SkillToResponse request config = %+v, want converted transport config", requestConfig)
	}
}
