package agentconfig

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

// ListAgentConfigsLogic 包含查询智能体配置列表用例。
type ListAgentConfigsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListAgentConfigsLogic 创建智能体配置列表用例。
func NewListAgentConfigsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAgentConfigsLogic {
	return &ListAgentConfigsLogic{ctx: ctx, svcCtx: svcCtx, log: xlog.Component("logic.agentconfig.listagentconfigs")}
}

// ListAgentConfigs 返回当前用户拥有的全部智能体配置。
func (l *ListAgentConfigsLogic) ListAgentConfigs(userID uuid.UUID) (*response.ListResponse[*response.AgentConfigResponse], error) {
	rows, err := l.svcCtx.AgentConfigRepo.List(l.ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.AgentConfigResponse, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			out = append(out, mapper.AgentConfigToResponse(row))
		}
	}
	return &response.ListResponse[*response.AgentConfigResponse]{List: out}, nil
}
