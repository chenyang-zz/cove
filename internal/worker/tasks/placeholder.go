package tasks

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/observability/xlog"
)

type PlaceholderTask struct {
	log *slog.Logger
}

func NewPlaceholderTask() *PlaceholderTask {
	return &PlaceholderTask{log: xlog.Component("worker.tasks.placeholder")}
}

func (h *PlaceholderTask) Handle(ctx context.Context, task *domain.Task) error {
	if h != nil && h.log != nil && task != nil {
		h.log.InfoContext(ctx, "任务 handler 暂未实现", slog.String("type", string(task.Name)))
	}
	return nil
}
