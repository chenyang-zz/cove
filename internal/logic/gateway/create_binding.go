package gateway

import (
	"context"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"log/slog"
)

// CreateBindingLogic contains the createBinding use case.
type CreateBindingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewCreateBindingLogic creates a CreateBindingLogic.
func NewCreateBindingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateBindingLogic {
	return &CreateBindingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.createbinding"),
	}
}

// CreateBinding 创建私聊或白名单群聊绑定
func (l *CreateBindingLogic) CreateBinding(userID uuid.UUID, input *request.CreateChannelBindingRequest) (*response.ChannelBindingResponse, error) {
	if input == nil {
		return nil, xerr.BadRequest("渠道绑定参数不能为空")
	}
	if _, err := l.svcCtx.ChannelGatewayRepo.FindAccountByID(l.ctx, userID, input.AccountID); err != nil {
		return nil, err
	}
	if input.AgentConfigID != nil {
		if _, err := l.svcCtx.AgentConfigRepo.FindByID(l.ctx, userID, *input.AgentConfigID); err != nil {
			return nil, err
		}
	}
	requireMention := input.ChatType == string(corechannel.ChatTypeGroup)
	toolPolicy := models.ChannelToolPolicyInherit
	if input.ChatType == string(corechannel.ChatTypeGroup) {
		toolPolicy = models.ChannelToolPolicySafe
	}
	enabled := true
	if input.RequireMention != nil {
		requireMention = *input.RequireMention
	}
	if input.ToolPolicy != nil {
		toolPolicy = *input.ToolPolicy
	}
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	routeKey := corechannel.StableRouteKey(input.AccountID.String(), input.ExternalChatID, input.ExternalThreadID)
	row, err := l.svcCtx.ChannelGatewayRepo.CreateBinding(l.ctx, userID, &models.ChannelBinding{
		AccountID: input.AccountID, RouteKey: routeKey, ChatType: input.ChatType,
		ExternalChatID: input.ExternalChatID, ExternalThreadID: input.ExternalThreadID,
		AgentConfigID: input.AgentConfigID, RequireMention: requireMention, ToolPolicy: toolPolicy, Enabled: enabled,
	})
	if err != nil {
		return nil, err
	}
	return bindingResponse(row), nil
}
