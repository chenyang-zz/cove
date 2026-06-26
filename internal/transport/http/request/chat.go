package request

type ChatStreamRequest struct {
	Message         string   `json:"message" binding:"required"`
	ConversationID  string   `json:"conversation_id"`
	EnableKnowledge *bool    `json:"enable_knowledge"`
	EnableMemory    *bool    `json:"enable_memory"`
	EnableWebSearch *bool    `json:"enable_web_search"`
	ImageKeys       []string `json:"image_keys"`
}
