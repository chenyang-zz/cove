/*
 * @Time   : 2026-07-10 11:51:43
 * @Author : chenyang
 * @File   : tool_config.go
 */

package models

import (
	"time"

	"github.com/google/uuid"
)

type ToolConfig struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`
	ToolKey   string    `gorm:"column:tool_key;type:varchar(128);not null;index"`             // 内置工具 key
	ToolType  string    `gorm:"column:tool_type;type:varchar(16);not null;default='builtin'"` // 工具类型 builtin | mcp
	Enabled   bool      `gorm:"column:enabled;type:boolean;not null;default:true"`            // 是否启用
	Config    JSONMap   `gorm:"column:config;type:jsonb;"`                                    // 工具特定配置 (预留)
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ToolConfig) TableName() string {
	return "tool_configs"
}
