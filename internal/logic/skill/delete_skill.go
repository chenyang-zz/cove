package skill

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
)

// DeleteSkillLogic contains the deleteSkill use case.
type DeleteSkillLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDeleteSkillLogic creates a DeleteSkillLogic.
func NewDeleteSkillLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSkillLogic {
	return &DeleteSkillLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.skill.deleteskill"),
	}
}

// DeleteSkill 删除skill
func (l *DeleteSkillLogic) DeleteSkill(userID uuid.UUID, input *request.UriSkillIDRequest) error {
	skillID, err := skillIDFromInput(input)
	if err != nil {
		return err
	}
	if err := l.svcCtx.SkillRepo.Delete(l.ctx, userID, skillID); err != nil {
		return err
	}
	l.log.InfoContext(l.ctx, "删除技能",
		slog.String("user_id", userID.String()),
		slog.String("skill_id", skillID.String()),
	)
	return nil
}
