/**
 * @Time   : 2026/7/9 18:43
 * @Author : chenyangzhao542@gmail.com
 * @File   : skill.go
 **/

package request

type FewShot struct {
	Input  string `json:"input" binding:"max=2000"`
	Output string `json:"output" binding:"max=2000"`
}

type SkillConfig struct {
	QuickPrompt []string  `json:"quick_prompt" binding:"omitempty,dive"`
	FewShots    []FewShot `json:"few_shots" binding:"omitempty,dive"`
}

type CreateSkillRequest struct {
	Name        string       `json:"name" binding:"required,min=1,max=64"`
	Description string       `json:"description" binding:"max=256"`
	Icon        *string      `json:"icon" binding:"omitempty,max=16"` // default 🧩
	Prompt      string       `json:"prompt" binding:"max=8000"`
	ToolKeys    []string     `json:"tool_keys" binding:"omitempty"`
	KBID        *string      `json:"kb_id" binding:"omitempty,uuid"`
	Enabled     *bool        `json:"enabled" binding:"omitempty"` // default true
	Config      *SkillConfig `json:"config" binding:"omitempty"`
}

type UriSkillIDRequest struct {
	ID string `uri:"skill_id" binding:"required,uuid"`
}

type UpdateSkillRequest struct {
	UriSkillIDRequest
	Name        *string      `json:"name" binding:"omitempty,min=1,max=64"`
	Description *string      `json:"description" binding:"omitempty,max=256"`
	Icon        *string      `json:"icon" binding:"omitempty,max=16"` // default 🧩
	Prompt      *string      `json:"prompt" binding:"omitempty,max=8000"`
	ToolKeys    []string     `json:"tool_keys" binding:"omitempty"`
	KBID        *string      `json:"kb_id" binding:"omitempty,uuid"`
	Enabled     *bool        `json:"enabled" binding:"omitempty"` // default true
	Config      *SkillConfig `json:"config" binding:"omitempty"`
}

type OptimizeSkillPromptRequest struct {
	Prompt string `json:"prompt" binding:"required,max=8000"`
}
