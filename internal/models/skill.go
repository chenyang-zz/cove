/**
 * @Time   : 2026/6/29 16:02
 * @Author : chenyangzhao542@gmail.com
 * @File   : skill.go
 **/

package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SkillFewShot 表示技能配置中的一组输入输出示例。
type SkillFewShot struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

// SkillConfig 表示技能的轻量 JSONB 配置。
type SkillConfig struct {
	QuickPrompt []string       `json:"quick_prompt,omitempty"`
	FewShots    []SkillFewShot `json:"few_shots,omitempty"`
}

// Value 将技能配置编码为 PostgreSQL JSONB 值；空配置编码为 JSON 对象。
func (c SkillConfig) Value() (driver.Value, error) {
	if len(c.QuickPrompt) == 0 && len(c.FewShots) == 0 {
		return "{}", nil
	}
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

// Scan 从 PostgreSQL JSONB 值解码技能配置；nil 会重置为零值。
func (c *SkillConfig) Scan(value any) error {
	if c == nil {
		return fmt.Errorf("SkillConfig scan target is nil")
	}
	switch v := value.(type) {
	case nil:
		*c = SkillConfig{}
		return nil
	case []byte:
		return json.Unmarshal(v, c)
	case string:
		return json.Unmarshal([]byte(v), c)
	default:
		return fmt.Errorf("unsupported SkillConfig scan type %T", value)
	}
}

type Skill struct {
	ID          uuid.UUID   `gorm:"column:id;type:uuid;primaryKey"`
	UserID      uuid.UUID   `gorm:"column:user_id;type:uuid;not null;index"`
	Name        string      `gorm:"column:name;size:64;not null"`
	Description string      `gorm:"column:description;size:256;default:''"`
	Icon        string      `gorm:"column:icon;size:16;default:'🧩'"`
	Prompt      string      `gorm:"column:prompt;type:text;default:''"`       // 专属任务提示词，对话时与角色卡 system_prompt 叠加注入
	ToolKeys    StringList  `gorm:"column:tool_keys;type:jsonb;default:'[]'"` // 工具白名单：内置工具 key 列表。非空=只用这些；空列表=不限定（用全局工具配置）
	KBID        *uuid.UUID  `gorm:"column:kb_id;type:uuid;index"`
	Config      SkillConfig `gorm:"column:config;type:jsonb;default:'{}'"`
	Enabled     bool        `gorm:"column:enabled;not null;default:true"`     // 是否在对话页技能选择器中显示（关闭则不占用对话框入口，避免技能多时拥挤）
	IsBuiltin   bool        `gorm:"column:is_builtin;not null;default:false"` // 是否由内置模板复制而来（标记用途，用户仍可改删）
	Sort        int         `gorm:"column:sort;not null;default:0"`           // 列表排序
	CreatedAt   time.Time   `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time   `gorm:"column:updated_at;autoUpdateTime"`

	User User          `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	KB   KnowledgeBase `gorm:"foreignKey:KBID;references:ID;constraint:OnDelete:SET NULL"`
}

func (Skill) TableName() string {
	return "skills"
}
