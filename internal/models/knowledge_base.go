/**
 * @Time   : 2026/6/29 16:09
 * @Author : chenyangzhao542@gmail.com
 * @File   : knowledge_base.go
 **/

// KnowledgeBase ORM 模型 —— PostgreSQL knowledge_bases 表（知识库分类）。
//
// 一个知识库 = 一组资料（文档 + 图片）的归属容器 + 检索范围。
// documents / images 通过 kb_id 归属到某个库；对话时可限定只检索某个库。
// 每个用户有一个 is_default=True 的默认库（不可删），存量与未指定归属的资料落入默认库。
// 文档/图片计数在读取时实时统计，不在本表冗余存储，避免计数漂移。

package models

import (
	"time"

	"github.com/google/uuid"
)

type KnowledgeBase struct {
	ID          uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	UserID      uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`
	User        User      `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Name        string    `gorm:"column:name;size:128;not null"`
	Description string    `gorm:"column:description;size:512"`
	Icon        string    `gorm:"column:icon;size:16"`
	Color       string    `gorm:"column:color;size:16;default:''"`
	IsDefault   bool      `gorm:"column:is_default;not null;default:false;index"`
	ChatEnabled bool      `gorm:"column:chat_enabled;not null;default:false"` // 是否参与对话检索：对话时检索所有 chat_enabled=True 的库。默认库默认开，其余默认关。
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KnowledgeBase) TableName() string {
	return "knowledge_bases"
}
