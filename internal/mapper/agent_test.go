package mapper

import (
	"testing"

	corecontext "github.com/boxify/api-go/internal/core/context"
	"github.com/boxify/api-go/internal/models"
	"github.com/google/uuid"
)

// TestAgentConfigToResponseIncludesResourceID 验证多配置响应会携带资源 ID、名称、默认标记和正确拼写的跨会话字段值。
func TestAgentConfigToResponseIncludesResourceID(t *testing.T) {
	id := uuid.New()
	out := AgentConfigToResponse(&models.AgentConfig{ID: id, Name: "日常助手", EnableCrossSession: true, IsDefault: true})
	if out.ID != id || out.Name != "日常助手" || !out.EnableCrossSession || !out.IsDefault {
		t.Fatalf("AgentConfigToResponse() = %#v, want id %s, name and default fields", out, id)
	}
}

// TestAgentConfigToContextPolicyUsesDefaultsForLegacyRows 验证旧记录缺少上下文字段时会使用 32K 默认策略。
func TestAgentConfigToContextPolicyUsesDefaultsForLegacyRows(t *testing.T) {
	policy := AgentConfigToContextPolicy(&models.AgentConfig{})
	if policy.WindowTokens != corecontext.DefaultWindowTokens || !policy.Enabled {
		t.Fatalf("AgentConfigToContextPolicy(legacy) = %#v, want enabled default policy", policy)
	}
}

// TestAgentConfigToContextPolicyMapsPersistedColumns 验证数据库独立列会完整映射为 core 策略。
func TestAgentConfigToContextPolicyMapsPersistedColumns(t *testing.T) {
	row := &models.AgentConfig{
		ContextEnabled: false, ContextWindowTokens: 8192, ContextOutputReserveTokens: 1024,
		ContextSafetyMarginTokens: 256, ContextTriggerRatio: 0.9, ContextTargetRatio: 0.7,
		ContextKeepRecentTokens: 2048, ContextSummaryMaxTokens: 512,
	}
	policy := AgentConfigToContextPolicy(row)
	if policy.Enabled || policy.WindowTokens != 8192 || policy.SummaryMaxTokens != 512 {
		t.Fatalf("AgentConfigToContextPolicy() = %#v, want persisted values", policy)
	}
}
