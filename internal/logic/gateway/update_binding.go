package gateway

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

// UpdateBindingLogic contains the updateBinding use case.
type UpdateBindingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewUpdateBindingLogic creates a UpdateBindingLogic.
func NewUpdateBindingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateBindingLogic {
	return &UpdateBindingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.updatebinding"),
	}
}

// UpdateBinding 更新绑定的 Agent、提及门控或工具策略
func (l *UpdateBindingLogic) UpdateBinding(userID uuid.UUID, input *request.UpdateChannelBindingDocRequest) (*response.ChannelBindingResponse, error) {
	if input == nil {
		return nil, xerr.BadRequest("渠道绑定参数不能为空")
	}
	bindingID, err := gatewayID(&input.UriGatewayIDRequest)
	if err != nil {
		return nil, err
	}
	values := make(map[string]any)
	if input.AgentConfigID != nil {
		if _, err := l.svcCtx.AgentConfigRepo.FindByID(l.ctx, userID, *input.AgentConfigID); err != nil {
			return nil, err
		}
		values["agent_config_id"] = *input.AgentConfigID
	} else if input.ClearAgentConfig {
		values["agent_config_id"] = nil
	}
	if input.RequireMention != nil {
		values["require_mention"] = *input.RequireMention
	}
	if input.ToolPolicy != nil {
		values["tool_policy"] = *input.ToolPolicy
	}
	if input.Enabled != nil {
		values["enabled"] = *input.Enabled
	}
	if len(values) == 0 {
		return nil, xerr.BadRequest("更新字段不能为空")
	}
	row, err := l.svcCtx.ChannelGatewayRepo.UpdateBinding(l.ctx, userID, bindingID, values)
	if err != nil {
		return nil, err
	}
	return bindingResponse(row), nil
}
