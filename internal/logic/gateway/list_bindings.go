package gateway

import (
	"context"
	"strings"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"log/slog"
)

// ListBindingsLogic contains the listBindings use case.
type ListBindingsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListBindingsLogic creates a ListBindingsLogic.
func NewListBindingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListBindingsLogic {
	return &ListBindingsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.listbindings"),
	}
}

// ListBindings 查询当前用户的确定性渠道路由绑定
func (l *ListBindingsLogic) ListBindings(userID uuid.UUID, input *request.ListChannelBindingsRequest) ([]*response.ChannelBindingResponse, error) {
	var accountID *uuid.UUID
	if input != nil && strings.TrimSpace(input.AccountID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(input.AccountID))
		if err != nil {
			return nil, xerr.BadRequest("渠道账号 ID 无效")
		}
		if _, err := l.svcCtx.ChannelGatewayRepo.FindAccountByID(l.ctx, userID, parsed); err != nil {
			return nil, err
		}
		accountID = &parsed
	}
	rows, err := l.svcCtx.ChannelGatewayRepo.ListBindings(l.ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ChannelBindingResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, bindingResponse(row))
	}
	return out, nil
}
