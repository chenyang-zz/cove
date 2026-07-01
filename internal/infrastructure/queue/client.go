package queue

import (
	"context"

	"github.com/boxify/api-go/internal/domain"
)

type EnqueueOptions struct {
	Queue    domain.QueueName
	MaxRetry *int
}

type EnqueueOption func(*EnqueueOptions)

type TaskInfo struct {
	ID    string
	Name  domain.TaskName
	Queue domain.QueueName
}

type Producer interface {
	Enqueue(ctx context.Context, task *domain.Task, opts ...EnqueueOption) (*TaskInfo, error)
	Close() error
}

type Handler interface {
	HandleTask(ctx context.Context, task *domain.Task) error
}

type HandlerFunc func(ctx context.Context, task *domain.Task) error

func (f HandlerFunc) HandleTask(ctx context.Context, task *domain.Task) error {
	return f(ctx, task)
}

type Router interface {
	Handle(name domain.TaskName, handler Handler)
}

func WithQueue(queue domain.QueueName) EnqueueOption {
	return func(opts *EnqueueOptions) {
		opts.Queue = queue
	}
}

func WithMaxRetry(maxRetry int) EnqueueOption {
	return func(opts *EnqueueOptions) {
		opts.MaxRetry = &maxRetry
	}
}

func NewEnqueueOptions(task *domain.Task, opts ...EnqueueOption) *EnqueueOptions {
	options := &EnqueueOptions{}
	if task != nil {
		options.Queue = task.Queue
	}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	return options
}
