// Package channel 定义外部消息渠道与 Cove 之间的业务无关契约。
package channel

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ProviderName 是稳定的渠道类型标识。
type ProviderName string

const (
	ProviderTelegram ProviderName = "telegram"
	ProviderFeishu   ProviderName = "feishu"
	ProviderWebhook  ProviderName = "webhook"
)

// ChatType 描述外部消息发生在私聊还是群聊。
type ChatType string

const (
	ChatTypeDirect ChatType = "direct"
	ChatTypeGroup  ChatType = "group"
)

// DeliveryState 描述平台发送结果能否安全重试。
type DeliveryState string

const (
	DeliverySent      DeliveryState = "sent"
	DeliveryTemporary DeliveryState = "temporary_error"
	DeliveryPermanent DeliveryState = "permanent_error"
	DeliveryUnknown   DeliveryState = "unknown"
)

// Capabilities 是 Provider 对外声明的能力矩阵。
type Capabilities struct {
	DirectMessages bool `json:"direct_messages"` // Provider 是否支持私聊消息。
	GroupMessages  bool `json:"group_messages"`  // Provider 是否支持群聊消息。
	Threads        bool `json:"threads"`         // Provider 是否支持线程消息。
	Replies        bool `json:"replies"`         // Provider 是否支持回复消息。
	Mentions       bool `json:"mentions"`        // Provider 是否支持 @ 提及。
	Typing         bool `json:"typing"`          // Provider 是否支持输入状态。
	InboundImages  bool `json:"inbound_images"`  // Provider 是否支持接收图片。
	InboundFiles   bool `json:"inbound_files"`   // Provider 是否支持接收文件。
	OutboundText   bool `json:"outbound_text"`   // Provider 是否支持发送文本。
}

// FieldDescriptor 描述未来管理界面可自动渲染的配置字段。
type FieldDescriptor struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Sensitive   bool   `json:"sensitive"` // 字段值是否敏感，前端应隐藏输入。
	Description string `json:"description,omitempty"`
}

// ProviderDescriptor 是 Provider 的静态元数据。
type ProviderDescriptor struct {
	Name             ProviderName      `json:"name"`
	DisplayName      string            `json:"display_name"`
	Description      string            `json:"description"`
	CredentialFields []FieldDescriptor `json:"credential_fields"` // Provider 连接所需的凭据字段。
	SettingFields    []FieldDescriptor `json:"setting_fields"`    // Provider 连接所需的可选设置字段。
	Capabilities     Capabilities      `json:"capabilities"`      // Provider 对外声明的能力矩阵。
	MaxTextLength    int               `json:"max_text_length"`   // Provider 支持的最大文本长度，超过该长度的消息应被截断。
}

// Validate 校验 Provider 描述是否足以安全注册。
func (d ProviderDescriptor) Validate() error {
	if strings.TrimSpace(string(d.Name)) == "" {
		return errors.New("provider name is required")
	}
	if strings.TrimSpace(d.DisplayName) == "" {
		return errors.New("provider display name is required")
	}
	if !d.Capabilities.OutboundText {
		return errors.New("provider must support outbound text")
	}
	if d.MaxTextLength <= 0 {
		return errors.New("provider max text length must be positive")
	}
	seen := make(map[string]struct{}, len(d.CredentialFields)+len(d.SettingFields))
	for _, field := range append(append([]FieldDescriptor(nil), d.CredentialFields...), d.SettingFields...) {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			return errors.New("provider field key is required")
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("provider field %q is duplicated", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// AccountConfig 是适配器启动和发送所需的已解密配置快照。
type AccountConfig struct {
	ID          string
	PublicID    string
	Credentials map[string]string
	Settings    map[string]any
}

// Route 唯一定位一个外部聊天或线程。
type Route struct {
	AccountID string   `json:"account_id"`
	ChatType  ChatType `json:"chat_type"`
	ChatID    string   `json:"chat_id"`
	ThreadID  string   `json:"thread_id,omitempty"`
}

// Key 返回不包含可逆用户信息的确定性路由键。
func (r Route) Key() string {
	return StableRouteKey(r.AccountID, r.ChatID, r.ThreadID)
}

// SenderIdentity 描述外部消息发送者。
type SenderIdentity struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
	Username    string `json:"username,omitempty"`
	IsBot       bool   `json:"is_bot"`
}

// ReplyReference 描述被回复的平台消息。
type ReplyReference struct {
	MessageID string `json:"message_id"`
	Text      string `json:"text,omitempty"`
}

// MediaReference 描述待受控下载的外部媒体。
type MediaReference struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	MIMEType string `json:"mime_type,omitempty"`
	FileName string `json:"file_name,omitempty"`
	Size     int64  `json:"size,omitempty"`
	URL      string `json:"url,omitempty"`
}

// InboundEvent 是所有 Provider 统一产出的规范化事件。
type InboundEvent struct {
	ID                string           `json:"id"`
	Provider          ProviderName     `json:"provider"`
	ProviderEventID   string           `json:"provider_event_id"`
	Route             Route            `json:"route"`
	Sender            SenderIdentity   `json:"sender"`
	PlatformMessageID string           `json:"platform_message_id,omitempty"`
	Text              string           `json:"text,omitempty"`
	Reply             *ReplyReference  `json:"reply,omitempty"`
	Mentioned         bool             `json:"mentioned"`
	Media             []MediaReference `json:"media,omitempty"`
	OccurredAt        time.Time        `json:"occurred_at"`
	ReceivedAt        time.Time        `json:"received_at"`
	Raw               []byte           `json:"-"`
}

// Validate 校验规范化事件的幂等和路由字段。
func (e InboundEvent) Validate() error {
	if strings.TrimSpace(e.ProviderEventID) == "" {
		return errors.New("provider event id is required")
	}
	if strings.TrimSpace(e.Route.AccountID) == "" || strings.TrimSpace(e.Route.ChatID) == "" {
		return errors.New("account id and chat id are required")
	}
	if e.Route.ChatType != ChatTypeDirect && e.Route.ChatType != ChatTypeGroup {
		return errors.New("chat type is invalid")
	}
	if strings.TrimSpace(e.Sender.ID) == "" {
		return errors.New("sender id is required")
	}
	if strings.TrimSpace(e.Text) == "" && len(e.Media) == 0 {
		return errors.New("event text or media is required")
	}
	return nil
}

// OutboundMessage 是数据面提交给 Provider 的最终消息。
type OutboundMessage struct {
	DeliveryID string          `json:"delivery_id"`
	Route      Route           `json:"route"`
	Text       string          `json:"text"`
	ReplyTo    *ReplyReference `json:"reply_to,omitempty"`
}

// Receipt 是 Provider 对一次发送尝试的确定性结论。
type Receipt struct {
	DeliveryID        string        `json:"delivery_id"`
	State             DeliveryState `json:"state"`
	PlatformMessageID string        `json:"platform_message_id,omitempty"`
	ErrorCode         string        `json:"error_code,omitempty"`
	ErrorMessage      string        `json:"error_message,omitempty"`
	RetryAfter        time.Duration `json:"-"`
}
