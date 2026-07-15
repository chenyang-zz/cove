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

// CreateAgentConfigLogic 包含创建智能体配置用例。
type CreateAgentConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewCreateAgentConfigLogic 创建智能体配置用例。
func NewCreateAgentConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAgentConfigLogic {
	return &CreateAgentConfigLogic{ctx: ctx, svcCtx: svcCtx, log: xlog.Component("logic.agentconfig.createagentconfig")}
}

// CreateAgentConfig 使用显式默认值和可选输入创建一条用户配置。
func (l *CreateAgentConfigLogic) CreateAgentConfig(userID uuid.UUID, input *request.CreateAgentConfigRequest) (*response.AgentConfigResponse, error) {
	config := defaultAgentConfig()
	if input != nil {
		applyAgentConfigFields(config, &input.AgentConfigFieldsRequest, nil)
		if input.Name != nil && config.Name == "" {
			return nil, xerr.BadRequest("智能体配置名称不能为空")
		}
	}
	if err := validateAgentConfig(config); err != nil {
		return nil, err
	}
	if l.svcCtx.AgentConfigRepo == nil {
		return nil, xerr.Internal("智能体配置仓储未初始化", nil)
	}
	config, err := l.svcCtx.AgentConfigRepo.Create(l.ctx, userID, config)
	if err != nil {
		return nil, err
	}
	return mapper.AgentConfigToResponse(config), nil
}
