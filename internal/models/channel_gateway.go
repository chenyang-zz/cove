package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ChannelAccountStatusUnknown  = "unknown"
	ChannelAccountStatusHealthy  = "healthy"
	ChannelAccountStatusDegraded = "degraded"

	ChannelAccountStatusDisabled = "disabled"
	ChannelIdentityStatusPending = "pending"
	ChannelIdentityStatusAllowed = "allowed"
	ChannelIdentityStatusBlocked = "blocked"

	ChannelInboxStatusReceived    = "received"
	ChannelInboxStatusIgnored     = "ignored"
	ChannelInboxStatusPendingPair = "pending_pairing"
	ChannelInboxStatusQueued      = "queued"
	ChannelInboxStatusProcessing  = "processing"
	ChannelInboxStatusCompleted   = "completed"
	ChannelInboxStatusFailed      = "failed"

	ChannelOutboxStatusPending = "pending"
	ChannelOutboxStatusSending = "sending"
	ChannelOutboxStatusRetry   = "retry"
	ChannelOutboxStatusSent    = "sent"
	ChannelOutboxStatusFailed  = "failed"
	ChannelOutboxStatusUnknown = "unknown"

	ChannelToolPolicyInherit = "inherit"
	ChannelToolPolicySafe    = "safe"
	ChannelToolPolicyNone    = "none"
)

// ChannelAccount 保存一个用户拥有的外部渠道账号配置。
type ChannelAccount struct {
	ID                   uuid.UUID  `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:uq_channel_accounts_id_user_id,priority:1"`
	UserID               uuid.UUID  `gorm:"column:user_id;type:uuid;not null;index;uniqueIndex:uq_channel_accounts_id_user_id,priority:2"`
	Provider             string     `gorm:"column:provider;size:32;not null;index"`
	Name                 string     `gorm:"column:name;size:128;not null"`
	PublicID             string     `gorm:"column:public_id;size:64;not null;uniqueIndex"`
	EncryptedCredentials JSONMap    `gorm:"column:encrypted_credentials;type:jsonb;not null"`
	Settings             JSONMap    `gorm:"column:settings;type:jsonb;not null"`
	DefaultAgentConfigID *uuid.UUID `gorm:"column:default_agent_config_id;type:uuid;index"`
	Enabled              bool       `gorm:"column:enabled;not null;default:true"`
	Status               string     `gorm:"column:status;size:24;not null;default:'unknown'"`
	LastError            string     `gorm:"column:last_error;size:1024;not null;default:''"`
	LastSeenAt           *time.Time `gorm:"column:last_seen_at"`
	CreatedAt            time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;autoUpdateTime"`

	User               User         `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	DefaultAgentConfig *AgentConfig `gorm:"foreignKey:DefaultAgentConfigID;references:ID;constraint:OnDelete:SET NULL"`
}

// TableName 返回渠道账号表名。
func (ChannelAccount) TableName() string { return "channel_accounts" }

// ChannelIdentity 保存外部用户的配对和阻止状态。
type ChannelIdentity struct {
	ID                uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	AccountID         uuid.UUID  `gorm:"column:account_id;type:uuid;not null;uniqueIndex:uq_channel_identity"`
	ExternalUserID    string     `gorm:"column:external_user_id;size:255;not null;uniqueIndex:uq_channel_identity"`
	ExternalChatID    string     `gorm:"column:external_chat_id;size:255;not null;default:'';uniqueIndex:uq_channel_identity"`
	DisplayName       string     `gorm:"column:display_name;size:255;not null;default:''"`
	Status            string     `gorm:"column:status;size:24;not null;default:'pending';index"`
	PairingCodeHash   string     `gorm:"column:pairing_code_hash;size:64;not null;default:''"`
	PairingCodeMasked string     `gorm:"column:pairing_code_masked;size:16;not null;default:''"`
	PairingExpiresAt  *time.Time `gorm:"column:pairing_expires_at;index"`
	ApprovedAt        *time.Time `gorm:"column:approved_at"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime"`

	Account ChannelAccount `gorm:"foreignKey:AccountID;references:ID;constraint:OnDelete:CASCADE"`
}

// TableName 返回渠道身份表名。
func (ChannelIdentity) TableName() string { return "channel_identities" }

// ChannelBinding 将外部聊天/线程确定性绑定到 Cove Conversation。
type ChannelBinding struct {
	ID uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	// UserID 是供控制面直接过滤的租户键，并由 Account 组合外键保证与账号所有者一致。
	UserID           uuid.UUID     `gorm:"column:user_id;type:uuid;not null;index"`
	AccountID        uuid.UUID     `gorm:"column:account_id;type:uuid;not null;uniqueIndex:uq_channel_binding_route"`
	RouteKey         string        `gorm:"column:route_key;size:64;not null;uniqueIndex:uq_channel_binding_route"`
	ChatType         string        `gorm:"column:chat_type;size:16;not null"`
	ExternalChatID   string        `gorm:"column:external_chat_id;size:255;not null"`
	ExternalThreadID string        `gorm:"column:external_thread_id;size:255;not null;default:''"`
	ConversationID   *uuid.UUID    `gorm:"column:conversation_id;type:uuid;index"`
	Conversation     *Conversation `gorm:"foreignKey:ConversationID;references:ID;constraint:OnDelete:SET NULL"`
	AgentConfigID    *uuid.UUID    `gorm:"column:agent_config_id;type:uuid;index"`
	AgentConfig      *AgentConfig  `gorm:"foreignKey:AgentConfigID;references:ID;constraint:OnDelete:SET NULL"`
	RequireMention   bool          `gorm:"column:require_mention;not null;default:true"`
	ToolPolicy       string        `gorm:"column:tool_policy;size:16;not null;default:'safe'"`
	Enabled          bool          `gorm:"column:enabled;not null;default:false"`
	CreatedAt        time.Time     `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time     `gorm:"column:updated_at;autoUpdateTime"`

	Account ChannelAccount `gorm:"foreignKey:AccountID,UserID;references:ID,UserID;constraint:fk_channel_bindings_account_tenant,OnUpdate:RESTRICT,OnDelete:CASCADE"`
}

// TableName 返回渠道绑定表名。
func (ChannelBinding) TableName() string { return "channel_bindings" }

// ChannelInboxEvent 是规范化入站事件的持久化收件箱。
type ChannelInboxEvent struct {
	ID uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:uq_channel_inbox_events_scope,priority:1"`
	// UserID 是异步恢复和审计使用的租户快照，并由 Account 组合外键约束。
	UserID                uuid.UUID  `gorm:"column:user_id;type:uuid;not null;index;uniqueIndex:uq_channel_inbox_events_scope,priority:3"`
	AccountID             uuid.UUID  `gorm:"column:account_id;type:uuid;not null;uniqueIndex:uq_channel_inbox_event;uniqueIndex:uq_channel_inbox_events_scope,priority:2"`
	ProviderEventID       string     `gorm:"column:provider_event_id;size:255;not null;uniqueIndex:uq_channel_inbox_event"`
	RouteKey              string     `gorm:"column:route_key;size:64;not null;index"`
	NormalizedEvent       JSONMap    `gorm:"column:normalized_event;type:jsonb;not null"`
	Status                string     `gorm:"column:status;size:32;not null;default:'received';index"`
	ConversationID        *uuid.UUID `gorm:"column:conversation_id;type:uuid;index"`
	CoveMessageID         *uuid.UUID `gorm:"column:cove_message_id;type:uuid;index"`
	AssistantMessageID    *uuid.UUID `gorm:"column:assistant_message_id;type:uuid;index"`
	ResolvedAgentConfigID *uuid.UUID `gorm:"column:resolved_agent_config_id;type:uuid;index"`
	ToolPolicy            string     `gorm:"column:tool_policy;size:16;not null;default:'inherit'"`
	TaskID                string     `gorm:"column:task_id;size:255;not null;default:''"`
	ErrorMessage          string     `gorm:"column:error_message;size:1024;not null;default:''"`
	EnqueuedAt            *time.Time `gorm:"column:enqueued_at"`
	ProcessedAt           *time.Time `gorm:"column:processed_at"`
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt             time.Time  `gorm:"column:updated_at;autoUpdateTime"`

	Account ChannelAccount `gorm:"foreignKey:AccountID,UserID;references:ID,UserID;constraint:fk_channel_inbox_events_account_tenant,OnUpdate:RESTRICT,OnDelete:CASCADE"`
}

// TableName 返回渠道收件箱表名。
func (ChannelInboxEvent) TableName() string { return "channel_inbox_events" }

// ChannelOutboxMessage 是最终回复的可靠发件箱。
type ChannelOutboxMessage struct {
	ID uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	// UserID 与 AccountID、InboxEventID 共同形成不可跨租户的出站归属链。
	UserID             uuid.UUID  `gorm:"column:user_id;type:uuid;not null;index"`
	AccountID          uuid.UUID  `gorm:"column:account_id;type:uuid;not null;index"`
	InboxEventID       uuid.UUID  `gorm:"column:inbox_event_id;type:uuid;not null;uniqueIndex"`
	ConversationID     uuid.UUID  `gorm:"column:conversation_id;type:uuid;not null;index"`
	AssistantMessageID uuid.UUID  `gorm:"column:assistant_message_id;type:uuid;not null;index"`
	DeliveryID         string     `gorm:"column:delivery_id;size:64;not null;uniqueIndex"`
	Route              JSONMap    `gorm:"column:route;type:jsonb;not null"`
	ReplyTo            JSONMap    `gorm:"column:reply_to;type:jsonb"`
	Content            string     `gorm:"column:content;type:text;not null"`
	Status             string     `gorm:"column:status;size:24;not null;default:'pending';index"`
	AttemptCount       int        `gorm:"column:attempt_count;not null;default:0"`
	NextAttemptAt      *time.Time `gorm:"column:next_attempt_at;index"`
	PlatformMessageID  string     `gorm:"column:platform_message_id;size:255;not null;default:''"`
	Receipt            JSONMap    `gorm:"column:receipt;type:jsonb"`
	LastError          string     `gorm:"column:last_error;size:1024;not null;default:''"`
	SentAt             *time.Time `gorm:"column:sent_at"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime"`

	Account    ChannelAccount    `gorm:"foreignKey:AccountID,UserID;references:ID,UserID;constraint:fk_channel_outbox_messages_account_tenant,OnUpdate:RESTRICT,OnDelete:CASCADE"`
	InboxEvent ChannelInboxEvent `gorm:"foreignKey:InboxEventID,AccountID,UserID;references:ID,AccountID,UserID;constraint:fk_channel_outbox_messages_inbox_scope,OnUpdate:RESTRICT,OnDelete:CASCADE"`
}

// TableName 返回渠道发件箱表名。
func (ChannelOutboxMessage) TableName() string { return "channel_outbox_messages" }
