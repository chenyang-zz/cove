package main

import (
	"log/slog"
	"os"

	"github.com/boxify/api-go/internal/config"
	"github.com/boxify/api-go/internal/infrastructure/queue"
	"github.com/hibiken/asynq"
)

func main() {
	cfg := config.Load()
	scheduler := asynq.NewScheduler(asynq.RedisClientOpt{Addr: cfg.Redis.Addr}, nil)
	if _, err := scheduler.Register("@daily", asynq.NewTask(queue.TaskMemoryConsolidate, nil)); err != nil {
		slog.Error("register scheduled task", "error", err)
		os.Exit(1)
	}
	slog.Info("scheduler starting")
	if err := scheduler.Run(); err != nil {
		slog.Error("scheduler stopped", "error", err)
		os.Exit(1)
	}
}
