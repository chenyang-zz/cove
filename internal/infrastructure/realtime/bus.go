package realtime

import (
	"context"
	"sync"

	"github.com/boxify/api-go/internal/domain"
)

type Bus struct {
	mu          sync.Mutex
	subscribers map[string][]chan domain.AgentEvent
}

func NewBus() *Bus {
	return &Bus{subscribers: map[string][]chan domain.AgentEvent{}}
}

func (b *Bus) Subscribe(ctx context.Context, conversationID string) <-chan domain.AgentEvent {
	ch := make(chan domain.AgentEvent, 16)
	b.mu.Lock()
	b.subscribers[conversationID] = append(b.subscribers[conversationID], ch)
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		defer b.mu.Unlock()
		close(ch)
	}()
	return ch
}

func (b *Bus) Publish(conversationID string, event domain.AgentEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subscribers[conversationID] {
		select {
		case ch <- event:
		default:
		}
	}
}
