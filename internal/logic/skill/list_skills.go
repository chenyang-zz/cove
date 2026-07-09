package skill

import (
	"context"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
	"log/slog"
)

// ListSkillsLogic contains the listSkills use case.
type ListSkillsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListSkillsLogic creates a ListSkillsLogic.
func NewListSkillsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSkillsLogic {
	return &ListSkillsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.skill.listskills"),
	}
}

// ListSkills 查询skill列表
func (l *ListSkillsLogic) ListSkills(userID uuid.UUID) (*response.ListResponse[*response.SkillResponse], error) {
	rows, err := l.svcCtx.SkillRepo.List(l.ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapper.SkillsToListResponse(rows), nil
}
