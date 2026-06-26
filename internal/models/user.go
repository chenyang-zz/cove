package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	Username       string     `gorm:"column:username;size:64;uniqueIndex;not null"`
	Nickname       *string    `gorm:"column:nickname;size:64"`
	Email          *string    `gorm:"column:email;size:255;uniqueIndex"`
	Avatar         *string    `gorm:"column:avatar;size:512"`
	PasswordHash   string     `gorm:"column:password_hash;size:255;not null"`
	BriefingSeenAt *time.Time `gorm:"column:briefing_seen_at"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (User) TableName() string {
	return "users"
}
