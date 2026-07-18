package gateway

import (
	"context"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
	"log/slog"
)

// DeleteBindingLogic contains the deleteBinding use case.
type DeleteBindingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDeleteBindingLogic creates a DeleteBindingLogic.
func NewDeleteBindingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteBindingLogic {
	return &DeleteBindingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.deletebinding"),
	}
}

// DeleteBinding 删除渠道路由绑定
func (l *DeleteBindingLogic) DeleteBinding(userID uuid.UUID, input *request.UriGatewayIDRequest) (*response.GatewayStatusResponse, error) {
	bindingID, err := gatewayID(input)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.ChannelGatewayRepo.DeleteBinding(l.ctx, userID, bindingID); err != nil {
		return nil, err
	}
	return &response.GatewayStatusResponse{Status: "deleted"}, nil
}
