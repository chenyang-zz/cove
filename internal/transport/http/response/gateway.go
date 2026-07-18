package response

import (
	"time"

	"github.com/boxify/api-go/internal/models"
	"github.com/google/uuid"
)

// ChannelAccountResponse 是不包含明文凭据的渠道账号。
type ChannelAccountResponse struct {
	ID                   uuid.UUID         `json:"id"`
	Provider             string            `json:"provider"`
	Name                 string            `json:"name"`
	PublicID             string            `json:"public_id"`
	CredentialMasked     map[string]string `json:"credential_masked"`
	Settings             models.JSONMap    `json:"settings"`
	DefaultAgentConfigID *uuid.UUID        `json:"default_agent_config_id"`
	Enabled              bool              `json:"enabled"`
	Status               string            `json:"status"`
	LastError            string            `json:"last_error,omitempty"`
	LastSeenAt           *time.Time        `json:"last_seen_at"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

// ChannelPairingResponse 展示待配对外部身份，不暴露完整配对码。
type ChannelPairingResponse struct {
	ID                uuid.UUID  `json:"id"`
	AccountID         uuid.UUID  `json:"account_id"`
	ExternalUserID    string     `json:"external_user_id"`
	ExternalChatID    string     `json:"external_chat_id"`
	DisplayName       string     `json:"display_name"`
	Status            string     `json:"status"`
	PairingCodeMasked string     `json:"pairing_code_masked"`
	PairingExpiresAt  *time.Time `json:"pairing_expires_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ChannelBindingResponse 展示确定性外部路由策略。
type ChannelBindingResponse struct {
	ID               uuid.UUID  `json:"id"`
	AccountID        uuid.UUID  `json:"account_id"`
	RouteKey         string     `json:"route_key"`
	ChatType         string     `json:"chat_type"`
	ExternalChatID   string     `json:"external_chat_id"`
	ExternalThreadID string     `json:"external_thread_id"`
	ConversationID   *uuid.UUID `json:"conversation_id"`
	AgentConfigID    *uuid.UUID `json:"agent_config_id"`
	RequireMention   bool       `json:"require_mention"`
	ToolPolicy       string     `json:"tool_policy"`
	Enabled          bool       `json:"enabled"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// GatewayCapabilitiesResponse 是供控制面展示的 Provider 能力矩阵。
type GatewayCapabilitiesResponse struct {
	DirectMessages bool `json:"direct_messages"`
	GroupMessages  bool `json:"group_messages"`
	Threads        bool `json:"threads"`
	Replies        bool `json:"replies"`
	Mentions       bool `json:"mentions"`
	Typing         bool `json:"typing"`
	InboundImages  bool `json:"inbound_images"`
	InboundFiles   bool `json:"inbound_files"`
	OutboundText   bool `json:"outbound_text"`
}

// GatewayFieldDescriptorResponse 描述可由管理界面渲染的字段。
type GatewayFieldDescriptorResponse struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Sensitive   bool   `json:"sensitive"`
	Description string `json:"description,omitempty"`
}

// GatewayProviderResponse 展示编译进 Cove 的 Provider 描述。
type GatewayProviderResponse struct {
	Name             string                           `json:"name"`
	DisplayName      string                           `json:"display_name"`
	Description      string                           `json:"description"`
	CredentialFields []GatewayFieldDescriptorResponse `json:"credential_fields"`
	SettingFields    []GatewayFieldDescriptorResponse `json:"setting_fields"`
	Capabilities     GatewayCapabilitiesResponse      `json:"capabilities"`
	MaxTextLength    int                              `json:"max_text_length"`
}

// GatewayStatusResponse 表示无需额外数据的网关操作结果。
type GatewayStatusResponse struct {
	Status string `json:"status"`
}
