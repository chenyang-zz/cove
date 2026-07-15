package agentconfig

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// SetDefaultAgentConfigLogic 包含切换默认智能体配置用例。
type SetDefaultAgentConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewSetDefaultAgentConfigLogic 创建切换默认智能体配置用例。
func NewSetDefaultAgentConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetDefaultAgentConfigLogic {
	return &SetDefaultAgentConfigLogic{ctx: ctx, svcCtx: svcCtx, log: xlog.Component("logic.agentconfig.setdefault")}
}

// SetDefaultAgentConfig 将当前用户拥有的指定配置设为唯一默认配置。
func (l *SetDefaultAgentConfigLogic) SetDefaultAgentConfig(userID uuid.UUID, input *request.UriAgentConfigIDRequest) (*response.AgentConfigResponse, error) {
	id, err := agentConfigID(input)
	if err != nil {
		return nil, err
	}
	if l.svcCtx.AgentConfigRepo == nil {
		return nil, xerr.Internal("智能体配置仓储未初始化", nil)
	}
	config, err := l.svcCtx.AgentConfigRepo.SetDefault(l.ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return mapper.AgentConfigToResponse(config), nil
}
