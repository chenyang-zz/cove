package tasks

import (
	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/queue"
	"github.com/boxify/api-go/internal/svc"
)

type Registry struct {
	svcCtx *svc.ServiceContext
}

func NewRegistry(svcCtx *svc.ServiceContext) *Registry {
	return &Registry{svcCtx: svcCtx}
}

func (r *Registry) Register(router queue.Router) {
	if router == nil {
		return
	}
	for _, taskName := range domain.TaskNames() {
		switch taskName {
		case domain.TaskParseDocument:
			router.Handle(taskName, queue.HandlerFunc(NewParseDocumentTask(r.svcCtx).Handle))
		default:
			router.Handle(taskName, queue.HandlerFunc(NewPlaceholderTask().Handle))
		}
	}
}
