package main

import (
	"log/slog"
	"os"

	"github.com/boxify/api-go/internal/config"
	"github.com/boxify/api-go/internal/domain"
	queueredis "github.com/boxify/api-go/internal/infrastructure/queue/redis"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/hibiken/asynq"
)

func main() {
	cfg := config.Load()
	xlog.Configure(xlog.Config{
		Env:   cfg.App.Env,
		Level: slog.LevelInfo,
		Color: true,
	})
	scheduler := asynq.NewScheduler(queueredis.ClientOpt(queueredis.Config{
		Addr:     cfg.Redis.Addr,
		Username: cfg.Redis.Username,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}), nil)
	if _, err := scheduler.Register("@daily", asynq.NewTask(string(domain.TaskMemoryConsolidate), nil)); err != nil {
		slog.Error("register scheduled task", "error", err)
		os.Exit(1)
	}
	slog.Info("scheduler starting")
	if err := scheduler.Run(); err != nil {
		slog.Error("scheduler stopped", "error", err)
		os.Exit(1)
	}
}
