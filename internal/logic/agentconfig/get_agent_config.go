package agentconfig

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

// GetAgentConfigLogic contains the getAgentConfig use case.
type GetAgentConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewGetAgentConfigLogic creates a GetAgentConfigLogic.
func NewGetAgentConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAgentConfigLogic {
	return &GetAgentConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.agentconfig.getagentconfig"),
	}
}

// GetAgentConfig 按 ID 查询当前用户拥有的智能体配置。
func (l *GetAgentConfigLogic) GetAgentConfig(userID uuid.UUID, input *request.UriAgentConfigIDRequest) (*response.AgentConfigResponse, error) {
	id, err := agentConfigID(input)
	if err != nil {
		return nil, err
	}
	config, err := l.svcCtx.AgentConfigRepo.FindByID(l.ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return mapper.AgentConfigToResponse(config), nil
}
