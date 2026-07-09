package models

import (
	"encoding/json"
	"testing"
)

// TestSkillConfigValueEncodesStructuredJSON 验证技能配置会编码为稳定的 JSONB 结构，空配置编码为对象。
func TestSkillConfigValueEncodesStructuredJSON(t *testing.T) {
	config := SkillConfig{
		QuickPrompt: []string{"快速问题"},
		FewShots:    []SkillFewShot{{Input: "问题", Output: "答案"}},
	}
	value, err := config.Value()
	if err != nil {
		t.Fatalf("SkillConfig.Value error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(value.(string)), &got); err != nil {
		t.Fatalf("unmarshal SkillConfig.Value error = %v", err)
	}
	if got["quick_prompt"] == nil || got["few_shots"] == nil {
		t.Fatalf("SkillConfig.Value = %v, want quick_prompt and few_shots", got)
	}

	empty, err := (SkillConfig{}).Value()
	if err != nil {
		t.Fatalf("empty SkillConfig.Value error = %v", err)
	}
	if empty != "{}" {
		t.Fatalf("empty SkillConfig.Value = %q, want {}", empty)
	}
}

// TestSkillConfigScanSupportsPostgresJSONValues 验证技能配置可以扫描 PostgreSQL 返回的字节、字符串和空值。
func TestSkillConfigScanSupportsPostgresJSONValues(t *testing.T) {
	payload := `{"quick_prompt":["快速问题"],"few_shots":[{"input":"问题","output":"答案"}]}`
	for _, value := range []any{[]byte(payload), payload} {
		var config SkillConfig
		if err := config.Scan(value); err != nil {
			t.Fatalf("SkillConfig.Scan(%T) error = %v", value, err)
		}
		if len(config.QuickPrompt) != 1 || config.QuickPrompt[0] != "快速问题" ||
			len(config.FewShots) != 1 || config.FewShots[0].Input != "问题" || config.FewShots[0].Output != "答案" {
			t.Fatalf("SkillConfig.Scan(%T) = %+v, want decoded config", value, config)
		}
	}

	config := SkillConfig{QuickPrompt: []string{"旧值"}}
	if err := config.Scan(nil); err != nil {
		t.Fatalf("SkillConfig.Scan(nil) error = %v", err)
	}
	if len(config.QuickPrompt) != 0 || len(config.FewShots) != 0 {
		t.Fatalf("SkillConfig.Scan(nil) = %+v, want zero value", config)
	}
}
