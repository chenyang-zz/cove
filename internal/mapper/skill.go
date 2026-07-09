package mapper

import (
	"strings"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
)

// SkillConfigFromRequest 将技能配置请求转换为模型配置，并规范化文本内容。
func SkillConfigFromRequest(input *request.SkillConfig) models.SkillConfig {
	if input == nil {
		return models.SkillConfig{}
	}
	out := models.SkillConfig{
		QuickPrompt: make([]string, 0, len(input.QuickPrompt)),
		FewShots:    make([]models.SkillFewShot, 0, len(input.FewShots)),
	}
	for _, prompt := range input.QuickPrompt {
		prompt = strings.TrimSpace(prompt)
		if prompt != "" {
			out.QuickPrompt = append(out.QuickPrompt, prompt)
		}
	}
	for _, shot := range input.FewShots {
		in := strings.TrimSpace(shot.Input)
		output := strings.TrimSpace(shot.Output)
		if in == "" && output == "" {
			continue
		}
		out.FewShots = append(out.FewShots, models.SkillFewShot{Input: in, Output: output})
	}
	return out
}

// SkillConfigToResponse 将模型层技能配置转换为传输层配置。
func SkillConfigToResponse(input models.SkillConfig) *request.SkillConfig {
	if len(input.QuickPrompt) == 0 && len(input.FewShots) == 0 {
		return nil
	}
	out := &request.SkillConfig{
		QuickPrompt: append([]string(nil), input.QuickPrompt...),
		FewShots:    make([]request.FewShot, 0, len(input.FewShots)),
	}
	for _, shot := range input.FewShots {
		out.FewShots = append(out.FewShots, request.FewShot{
			Input:  shot.Input,
			Output: shot.Output,
		})
	}
	return out
}

func SkillToResponse(row *models.Skill) *response.SkillResponse {
	if row == nil {
		return nil
	}
	return &response.SkillResponse{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Icon:        row.Icon,
		Prompt:      row.Prompt,
		ToolKeys:    []string(row.ToolKeys),
		KBID:        row.KBID,
		Enabled:     row.Enabled,
		Config:      SkillConfigToResponse(row.Config),
		IsBuiltin:   row.IsBuiltin,
	}
}

func SkillsToListResponse(rows []*models.Skill) *response.ListResponse[*response.SkillResponse] {
	out := make([]*response.SkillResponse, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, SkillToResponse(row))
	}
	return &response.ListResponse[*response.SkillResponse]{List: out}
}
