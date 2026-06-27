/**
 * @Time   : 2026/6/27 19:45
 * @Author : chenyangzhao542@gmail.com
 * @File   : message.go
 **/

package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MessageMetaData struct {
	ImageKeys  []string `json:"image_keys"`
	SenderName string   `json:"sender_name"`
}

func (m MessageMetaData) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *MessageMetaData) Scan(value any) error {
	if value == nil {
		*m = MessageMetaData{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan MessageMetaData")
	}

	return json.Unmarshal(bytes, m)
}

type Message struct {
	ID             uuid.UUID    `gorm:"column:id;type:uuid;primaryKey"`
	ConversationID uuid.UUID    `gorm:"column:conversation_id;type:uuid;not null;index"`
	Conversation   Conversation `gorm:"foreignKey:ConversationID;references:ID;constraint:OnDelete:CASCADE"`
	Role           string       `gorm:"column:role;size:16;not null"`
	Content        string       `gorm:"column:content;type:text;not null"`
	// 群聊中该消息由哪个角色卡发出（user 消息为空；单聊 assistant 也为空）
	SenderPersonID uuid.UUID `gorm:"column:sender_person_id;type:uuid;"`
	// 多人实时群聊中该 user 消息由哪个真人发出（单人会话/AI 消息为空）
	SenderUserID uuid.UUID `gorm:"column:sender_user_id;type:uuid;"`
	// 附加信息：引用 citations / 工具调用 tool_calls / token usage / 图片等
	MetaData  *MessageMetaData `gorm:"column:meta_data;type:jsonb"`
	CreatedAt time.Time        `gorm:"column:created_at;autoCreateTime"`
}

func (Message) TableName() string {
	return "messages"
}
