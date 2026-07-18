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

// GetBindingLogic contains the getBinding use case.
type GetBindingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewGetBindingLogic creates a GetBindingLogic.
func NewGetBindingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBindingLogic {
	return &GetBindingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.getbinding"),
	}
}

// GetBinding 查询渠道路由绑定详情
func (l *GetBindingLogic) GetBinding(userID uuid.UUID, input *request.UriGatewayIDRequest) (*response.ChannelBindingResponse, error) {
	bindingID, err := gatewayID(input)
	if err != nil {
		return nil, err
	}
	row, err := l.svcCtx.ChannelGatewayRepo.FindBindingByID(l.ctx, userID, bindingID)
	if err != nil {
		return nil, err
	}
	return bindingResponse(row), nil
}
