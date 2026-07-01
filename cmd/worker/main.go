package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/boxify/api-go/internal/config"
	"github.com/boxify/api-go/internal/domain"
	queueredis "github.com/boxify/api-go/internal/infrastructure/queue/redis"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	workertasks "github.com/boxify/api-go/internal/worker/tasks"
	"github.com/hibiken/asynq"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	xlog.Configure(xlog.Config{
		Env:   cfg.App.Env,
		Level: slog.LevelInfo,
		Color: true,
	})
	svcCtx, err := svc.New(ctx, cfg)
	if err != nil {
		slog.Error("init service context", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := svcCtx.Close(context.Background()); err != nil {
			slog.Error("close service context", "error", err)
		}
	}()

	mux := asynq.NewServeMux()
	workertasks.NewRegistry(svcCtx).Register(queueredis.NewRouter(mux))
	server := asynq.NewServer(
		queueredis.ClientOpt(queueredis.Config{
			Addr:     cfg.Redis.Addr,
			Username: cfg.Redis.Username,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}),
		asynq.Config{Queues: map[string]int{
			string(domain.QueueDefault):  5,
			string(domain.QueueParse):    3,
			string(domain.QueueMemory):   3,
			string(domain.QueueResearch): 1,
			string(domain.QueueBeat):     1,
		}},
	)
	if err := server.Run(mux); err != nil {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
