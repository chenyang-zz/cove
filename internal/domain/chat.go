package domain

import "github.com/google/uuid"

type ChatStreamInput struct {
	UserID  uuid.UUID
	Message string
}

type ConversationMeta struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	Title          string    `json:"title"`
}
