package agentconfig

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// DeleteAgentConfigLogic 包含删除智能体配置用例。
type DeleteAgentConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDeleteAgentConfigLogic 创建删除智能体配置用例。
func NewDeleteAgentConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAgentConfigLogic {
	return &DeleteAgentConfigLogic{ctx: ctx, svcCtx: svcCtx, log: xlog.Component("logic.agentconfig.deleteagentconfig")}
}

// DeleteAgentConfig 删除当前用户拥有的指定智能体配置。
func (l *DeleteAgentConfigLogic) DeleteAgentConfig(userID uuid.UUID, input *request.UriAgentConfigIDRequest) error {
	id, err := agentConfigID(input)
	if err != nil {
		return err
	}
	if l.svcCtx.AgentConfigRepo == nil {
		return xerr.Internal("智能体配置仓储未初始化", nil)
	}
	return l.svcCtx.AgentConfigRepo.Delete(l.ctx, userID, id)
}
