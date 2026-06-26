package app

import (
	"context"
	"strings"

	"github.com/boxify/api-go/internal/domain"
	"github.com/google/uuid"
)

type ChatService struct{}

func NewChatService() *ChatService {
	return &ChatService{}
}

func (s *ChatService) Stream(ctx context.Context, input domain.ChatStreamInput) (<-chan domain.AgentEvent, error) {
	events := make(chan domain.AgentEvent, 4)
	go func() {
		defer close(events)
		conversationID := uuid.New()
		title := strings.TrimSpace(input.Message)
		if title == "" {
			title = "新对话"
		}
		if len([]rune(title)) > 20 {
			title = string([]rune(title)[:20])
		}
		select {
		case <-ctx.Done():
			return
		case events <- domain.AgentEvent{Type: "meta", Text: conversationID.String(), Stats: map[string]any{"title": title}}:
		}
		select {
		case <-ctx.Done():
			return
		case events <- domain.AgentEvent{Type: "token", Text: "hello"}:
		}
		select {
		case <-ctx.Done():
			return
		case events <- domain.AgentEvent{Type: "done", Text: conversationID.String()}:
		}
	}()
	return events, nil
}
