package queue_test

import (
	"context"
	"testing"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/queue"
	"github.com/google/uuid"
)

func TestHandlerFuncWrapsFunction(t *testing.T) {
	// 验证 HandlerFunc 能把普通函数适配成队列 handler，方便 worker task 独立注册。
	task, err := domain.NewParseDocumentTask(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}
	var called bool
	handler := queue.HandlerFunc(func(ctx context.Context, got *domain.Task) error {
		called = got == task
		return nil
	})

	if err := handler.HandleTask(context.Background(), task); err != nil {
		t.Fatalf("HandleTask error = %v", err)
	}
	if !called {
		t.Fatal("handler was not called with the domain task")
	}
}

func TestNewEnqueueOptionsUsesTaskQueueAndOverrides(t *testing.T) {
	// 验证通用入队 option 默认使用 domain task 队列，并支持调用方覆盖队列和重试次数。
	task, err := domain.NewParseDocumentTask(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("NewParseDocumentTask error = %v", err)
	}

	opts := queue.NewEnqueueOptions(task, queue.WithQueue(domain.QueueBeat), queue.WithMaxRetry(3))
	if opts.Queue != domain.QueueBeat {
		t.Fatalf("queue = %q, want %q", opts.Queue, domain.QueueBeat)
	}
	if opts.MaxRetry == nil || *opts.MaxRetry != 3 {
		t.Fatalf("max retry = %v, want 3", opts.MaxRetry)
	}
}
