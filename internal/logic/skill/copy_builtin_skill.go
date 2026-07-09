package skill

import (
	"context"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"log/slog"
)

// CopyBuiltinSkillLogic contains the copyBuiltinSkill use case.
type CopyBuiltinSkillLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewCopyBuiltinSkillLogic creates a CopyBuiltinSkillLogic.
func NewCopyBuiltinSkillLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CopyBuiltinSkillLogic {
	return &CopyBuiltinSkillLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.skill.copybuiltinskill"),
	}
}

// CopyBuiltinSkill 把内置技能复制为用户技能
func (l *CopyBuiltinSkillLogic) CopyBuiltinSkill(userID uuid.UUID, input *request.UriSkillIDRequest) (*response.SkillResponse, error) {
	if _, err := skillIDFromInput(input); err != nil {
		return nil, err
	}
	return nil, xerr.BadRequest("内置技能暂未实现")
}
