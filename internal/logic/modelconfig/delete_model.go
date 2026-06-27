package modelconfig

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
)

type DeleteModelLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

func NewDeleteModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteModelLogic {
	return &DeleteModelLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.modelconfig.deletemodel"),
	}
}

func (l *DeleteModelLogic) DeleteModel(userID uuid.UUID, input *request.UriConfigIDRequest) error {
	configID, err := configIDFromInput(input)
	if err != nil {
		return err
	}
	config, err := l.svcCtx.ModelConfigRepo.FindByID(l.ctx, userID, configID)
	if err != nil {
		return err
	}

	err = l.svcCtx.ModelConfigRepo.Delete(l.ctx, config.ID)
	if err != nil {
		return err
	}

	l.log.InfoContext(l.ctx, "删除模型配置",
		slog.String("user", userID.String()),
		slog.String("config_id", config.ID.String()),
	)
	return nil
}
