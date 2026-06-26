package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/boxify/api-go/internal/config"
	"github.com/boxify/api-go/internal/infrastructure/queue"
	"github.com/hibiken/asynq"
)

func main() {
	cfg := config.Load()
	mux := asynq.NewServeMux()
	for _, name := range queue.TaskNames() {
		taskName := name
		mux.HandleFunc(taskName, func(ctx context.Context, task *asynq.Task) error {
			slog.Info("task received", "type", task.Type())
			return nil
		})
	}
	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Addr},
		asynq.Config{Queues: map[string]int{
			queue.QueueDefault:  5,
			queue.QueueParse:    3,
			queue.QueueMemory:   3,
			queue.QueueResearch: 1,
			queue.QueueBeat:     1,
		}},
	)
	if err := server.Run(mux); err != nil {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
