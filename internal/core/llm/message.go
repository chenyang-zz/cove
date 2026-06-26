package llm

func UserMessage(content string) *Message {
	return &Message{
		Role:    UserRole,
		Content: content,
	}
}

func AssistantMessage(content string) *Message {
	return &Message{
		Role:    AssistantRole,
		Content: content,
	}
}

func SystemMessage(content string) *Message {
	return &Message{
		Role:    SystemRole,
		Content: content,
	}
}
