package llm

type MessageRoleType string

const (
	SystemRole    MessageRoleType = "system"
	UserRole      MessageRoleType = "user"
	AssistantRole MessageRoleType = "assistant"
)

type Message struct {
	Role    MessageRoleType `json:"role"`
	Content string          `json:"content"`
}
