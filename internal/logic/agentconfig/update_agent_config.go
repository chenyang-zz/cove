package agentconfig

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// UpdateAgentConfigLogic contains the updateAgentConfig use case.
type UpdateAgentConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewUpdateAgentConfigLogic creates a UpdateAgentConfigLogic.
func NewUpdateAgentConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAgentConfigLogic {
	return &UpdateAgentConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.agentconfig.updateagentconfig"),
	}
}

// UpdateAgentConfig 更新智能体配置
func (l *UpdateAgentConfigLogic) UpdateAgentConfig(userID uuid.UUID, input *request.UpdateAgentConfigRequest) (*response.AgentConfigResponse, error) {
	if input == nil {
		return nil, xerr.BadRequest("更新参数不能为空")
	}
	id, err := agentConfigID(&input.UriAgentConfigIDRequest)
	if err != nil {
		return nil, err
	}
	config, err := l.svcCtx.AgentConfigRepo.FindByID(l.ctx, userID, id)
	if err != nil {
		return nil, err
	}
	candidate := *config
	fields := repository.NewAgentConfigUpdateFields()
	applyAgentConfigFields(&candidate, &input.AgentConfigFieldsRequest, fields)
	if input.Name != nil && candidate.Name == "" {
		return nil, xerr.BadRequest("智能体配置名称不能为空")
	}
	if err := validateAgentConfig(&candidate); err != nil {
		return nil, err
	}

	config, err = l.svcCtx.AgentConfigRepo.UpdateFields(l.ctx, userID, id, &candidate, fields)
	if err != nil {
		return nil, err
	}

	return mapper.AgentConfigToResponse(config), nil
}
