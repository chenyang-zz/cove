package queue

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

type Producer struct {
	client *asynq.Client
}

func NewProducer(redisAddr string) *Producer {
	return &Producer{client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})}
}

func (p *Producer) Enqueue(ctx context.Context, taskName string, payload any, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	task := asynq.NewTask(taskName, data)
	return p.client.EnqueueContext(ctx, task, opts...)
}

func (p *Producer) Close() error {
	return p.client.Close()
}
