package agentconfig

import (
	"strings"

	corecontext "github.com/boxify/api-go/internal/core/context"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

func defaultAgentConfig() *models.AgentConfig {
	policy := corecontext.DefaultPolicy()
	return &models.AgentConfig{
		Temperature:                0.7,
		EnableKnowledge:            true,
		EnableMemory:               true,
		EnableActiveRecall:         true,
		ContextEnabled:             policy.Enabled,
		ContextWindowTokens:        policy.WindowTokens,
		ContextOutputReserveTokens: policy.OutputReserveTokens,
		ContextSafetyMarginTokens:  policy.SafetyMarginTokens,
		ContextTriggerRatio:        policy.TriggerRatio,
		ContextTargetRatio:         policy.TargetRatio,
		ContextKeepRecentTokens:    policy.KeepRecentTokens,
		ContextSummaryMaxTokens:    policy.SummaryMaxTokens,
	}
}

func applyAgentConfigFields(config *models.AgentConfig, input *request.AgentConfigFieldsRequest, fields *repository.AgentConfigUpdateFields) {
	if config == nil || input == nil {
		return
	}
	if input.Name != nil {
		config.Name = strings.TrimSpace(*input.Name)
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).Name)
	}
	if input.SystemPrompt != nil {
		config.SystemPrompt = *input.SystemPrompt
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).SystemPrompt)
	}
	if input.Temperature != nil {
		config.Temperature = *input.Temperature
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).Temperature)
	}
	if input.EnableKnowledge != nil {
		config.EnableKnowledge = *input.EnableKnowledge
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).EnableKnowledge)
	}
	if input.EnableMemory != nil {
		config.EnableMemory = *input.EnableMemory
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).EnableMemory)
	}
	if input.EnableWebSearch != nil {
		config.EnableWebSearch = *input.EnableWebSearch
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).EnableWebSearch)
	}
	if input.EnableActiveRecall != nil {
		config.EnableActiveRecall = *input.EnableActiveRecall
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).EnableActiveRecall)
	}
	if input.EnableCrossSession != nil {
		config.EnableCrossSession = *input.EnableCrossSession
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).EnableCrossSession)
	}
	if input.ShowAvatar != nil {
		config.ShowAvatar = *input.ShowAvatar
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).ShowAvatar)
	}
	if input.HumanMode != nil {
		config.HumanMode = *input.HumanMode
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).HumanMode)
	}
	if input.ContextEnabled != nil {
		config.ContextEnabled = *input.ContextEnabled
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).ContextEnabled)
	}
	if input.ContextWindowTokens != nil {
		config.ContextWindowTokens = *input.ContextWindowTokens
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).ContextWindowTokens)
	}
	if input.ContextOutputReserveTokens != nil {
		config.ContextOutputReserveTokens = *input.ContextOutputReserveTokens
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).ContextOutputReserveTokens)
	}
	if input.ContextSafetyMarginTokens != nil {
		config.ContextSafetyMarginTokens = *input.ContextSafetyMarginTokens
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).ContextSafetyMarginTokens)
	}
	if input.ContextTriggerRatio != nil {
		config.ContextTriggerRatio = *input.ContextTriggerRatio
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).ContextTriggerRatio)
	}
	if input.ContextTargetRatio != nil {
		config.ContextTargetRatio = *input.ContextTargetRatio
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).ContextTargetRatio)
	}
	if input.ContextKeepRecentTokens != nil {
		config.ContextKeepRecentTokens = *input.ContextKeepRecentTokens
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).ContextKeepRecentTokens)
	}
	if input.ContextSummaryMaxTokens != nil {
		config.ContextSummaryMaxTokens = *input.ContextSummaryMaxTokens
		addAgentConfigField(fields, (*repository.AgentConfigUpdateFields).ContextSummaryMaxTokens)
	}
}

func addAgentConfigField(fields *repository.AgentConfigUpdateFields, add func(*repository.AgentConfigUpdateFields) *repository.AgentConfigUpdateFields) {
	if fields != nil {
		add(fields)
	}
}

func validateAgentConfig(config *models.AgentConfig) error {
	if config.Name != "" && len([]rune(config.Name)) > 100 {
		return xerr.BadRequest("智能体配置名称不能超过 100 个字符")
	}
	if err := mapper.AgentConfigToContextPolicy(config).Validate(); err != nil {
		return xerr.BadRequest(err.Error())
	}
	return nil
}

func agentConfigID(input *request.UriAgentConfigIDRequest) (uuid.UUID, error) {
	if input == nil {
		return uuid.Nil, xerr.BadRequest("智能体配置 ID 不能为空")
	}
	id, err := uuid.Parse(input.AgentConfigID)
	if err != nil {
		return uuid.Nil, xerr.BadRequest("智能体配置 ID 无效")
	}
	return id, nil
}
