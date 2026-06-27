/**
 * @Time   : 2026/6/27 15:44
 * @Author : chenyangzhao542@gmail.com
 * @File   : conversation.go
 **/

package response

import (
	"time"

	"github.com/google/uuid"
)

type ConversationResponse struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	IsGroup         bool      `json:"is_group"`
	MemberPersonIDs []string  `json:"member_person_i_ds"`
	EnableTools     bool      `json:"enable_tools"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type MessageMetaData struct {
	ImageKeys  []string `json:"image_keys,omitempty"`
	SenderName string   `json:"sender_name,omitempty"`
}

type MessageResponse struct {
	ID             uuid.UUID        `json:"id"`
	Role           string           `json:"role"`
	Content        string           `json:"content"`
	MetaData       *MessageMetaData `json:"meta_data"`
	Images         []string         `json:"images"`
	SenderPersonID *uuid.UUID       `json:"sender_person_id"`
	SenderName     *string          `json:"sender_name"`
	Feedback       *string          `json:"feedback"`
	CreatedAt      time.Time        `json:"created_at"`
}
