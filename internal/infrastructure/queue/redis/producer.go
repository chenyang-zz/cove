package redis

import (
	"context"
	"fmt"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/queue"
	"github.com/hibiken/asynq"
)

type Producer struct {
	client *asynq.Client
}

func NewProducer(cfg Config) *Producer {
	return &Producer{client: asynq.NewClient(ClientOpt(cfg))}
}

func (p *Producer) Enqueue(ctx context.Context, task *domain.Task, opts ...queue.EnqueueOption) (*queue.TaskInfo, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("queue producer is nil")
	}
	asynqTask, err := EncodeTask(task)
	if err != nil {
		return nil, err
	}
	info, err := p.client.EnqueueContext(ctx, asynqTask, enqueueOptions(task, opts...)...)
	if err != nil {
		return nil, err
	}
	return &queue.TaskInfo{
		ID:    info.ID,
		Name:  domain.TaskName(info.Type),
		Queue: domain.QueueName(info.Queue),
	}, nil
}

func (p *Producer) Close() error {
	if p == nil || p.client == nil {
		return nil
	}
	return p.client.Close()
}

func enqueueOptions(task *domain.Task, opts ...queue.EnqueueOption) []asynq.Option {
	options := queue.NewEnqueueOptions(task, opts...)
	asynqOptions := make([]asynq.Option, 0, 2)
	if options.Queue != "" {
		asynqOptions = append(asynqOptions, asynq.Queue(string(options.Queue)))
	}
	if options.MaxRetry != nil {
		asynqOptions = append(asynqOptions, asynq.MaxRetry(*options.MaxRetry))
	}
	return asynqOptions
}
