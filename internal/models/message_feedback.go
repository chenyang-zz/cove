/**
 * @Time   : 2026/6/27 20:12
 * @Author : chenyangzhao542@gmail.com
 * @File   : message_feedback.go
 **/

package models

import (
	"time"

	"github.com/google/uuid"
)

type MessageFeedback struct {
	ID             uuid.UUID    `gorm:"column:id;type:uuid;primaryKey"`
	UserID         uuid.UUID    `gorm:"column:user_id;type:uuid;not null;index"`
	User           User         `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	MessageID      uuid.UUID    `gorm:"column:message_id;type:uuid;not null;index"`
	Message        Message      `gorm:"foreignKey:MessageID;references:ID;constraint:OnDelete:CASCADE"`
	ConversationID uuid.UUID    `gorm:"column:conversation_id;type:uuid;not null;index"`
	Conversation   Conversation `gorm:"foreignKey:ConversationID;references:ID;constraint:OnDelete:CASCADE"`
	Rating         string       `gorm:"column:rating;size:8;not null"` // up | down
	Comment        string       `gorm:"column:comment;type:text"`
	CreatedAt      time.Time    `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time    `gorm:"column:updated_at;autoUpdateTime"`
}

func (MessageFeedback) TableName() string {
	return "message_feedbacks"
}
