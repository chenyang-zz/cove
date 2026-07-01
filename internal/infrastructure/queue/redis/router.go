package redis

import (
	"context"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/queue"
	"github.com/hibiken/asynq"
)

type Router struct {
	mux *asynq.ServeMux
}

func NewRouter(mux *asynq.ServeMux) *Router {
	return &Router{mux: mux}
}

func (r *Router) Handle(name domain.TaskName, handler queue.Handler) {
	if r == nil || r.mux == nil || handler == nil {
		return
	}
	r.mux.HandleFunc(string(name), func(ctx context.Context, task *asynq.Task) error {
		domainTask, err := DecodeTask(task)
		if err != nil {
			return err
		}
		return handler.HandleTask(ctx, domainTask)
	})
}
