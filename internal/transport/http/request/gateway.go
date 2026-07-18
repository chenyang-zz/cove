package request

import "github.com/google/uuid"

// UriGatewayIDRequest 描述网关资源路径参数。
type UriGatewayIDRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

// CreateChannelAccountRequest 创建渠道账号。
type CreateChannelAccountRequest struct {
	Provider             string            `json:"provider" binding:"required,oneof=telegram feishu webhook"`
	Name                 string            `json:"name" binding:"required,min=1,max=128"`
	Credentials          map[string]string `json:"credentials" binding:"required"`
	Settings             map[string]any    `json:"settings"`
	DefaultAgentConfigID *uuid.UUID        `json:"default_agent_config_id"`
	Enabled              *bool             `json:"enabled"`
}

// UpdateChannelAccountRequest 更新渠道账号的可编辑配置。
type UpdateChannelAccountRequest struct {
	Name                 *string           `json:"name" binding:"omitempty,min=1,max=128"`
	Credentials          map[string]string `json:"credentials"`
	Settings             map[string]any    `json:"settings"`
	DefaultAgentConfigID *uuid.UUID        `json:"default_agent_config_id"`
	ClearDefaultAgent    bool              `json:"clear_default_agent"`
	Enabled              *bool             `json:"enabled"`
}

// UpdateChannelAccountDocRequest 组合路径参数和 JSON body，仅用于 OpenAPI 描述。
type UpdateChannelAccountDocRequest struct {
	UriGatewayIDRequest
	UpdateChannelAccountRequest
}

// ListChannelBindingsRequest 按渠道账号筛选绑定。
type ListChannelBindingsRequest struct {
	AccountID string `form:"account_id" json:"account_id" binding:"omitempty,uuid"`
}

// CreateChannelBindingRequest 创建外部路由绑定。
type CreateChannelBindingRequest struct {
	AccountID        uuid.UUID  `json:"account_id" binding:"required"`
	ChatType         string     `json:"chat_type" binding:"required,oneof=direct group"`
	ExternalChatID   string     `json:"external_chat_id" binding:"required,max=255"`
	ExternalThreadID string     `json:"external_thread_id" binding:"omitempty,max=255"`
	AgentConfigID    *uuid.UUID `json:"agent_config_id"`
	RequireMention   *bool      `json:"require_mention"`
	ToolPolicy       *string    `json:"tool_policy" binding:"omitempty,oneof=inherit safe none"`
	Enabled          *bool      `json:"enabled"`
}

// UpdateChannelBindingRequest 更新绑定策略。
type UpdateChannelBindingRequest struct {
	AgentConfigID    *uuid.UUID `json:"agent_config_id"`
	ClearAgentConfig bool       `json:"clear_agent_config"`
	RequireMention   *bool      `json:"require_mention"`
	ToolPolicy       *string    `json:"tool_policy" binding:"omitempty,oneof=inherit safe none"`
	Enabled          *bool      `json:"enabled"`
}

// UpdateChannelBindingDocRequest 组合路径参数和 JSON body，仅用于 OpenAPI 描述。
type UpdateChannelBindingDocRequest struct {
	UriGatewayIDRequest
	UpdateChannelBindingRequest
}
