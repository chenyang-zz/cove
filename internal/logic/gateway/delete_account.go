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

// DeleteAccountLogic contains the deleteAccount use case.
type DeleteAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDeleteAccountLogic creates a DeleteAccountLogic.
func NewDeleteAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAccountLogic {
	return &DeleteAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.deleteaccount"),
	}
}

// DeleteAccount 删除渠道账号及其网关数据
func (l *DeleteAccountLogic) DeleteAccount(userID uuid.UUID, input *request.UriGatewayIDRequest) (*response.GatewayStatusResponse, error) {
	accountID, err := gatewayID(input)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.ChannelGatewayRepo.DeleteAccount(l.ctx, userID, accountID); err != nil {
		return nil, err
	}
	publishReload(l.ctx, l.svcCtx, accountID)
	return &response.GatewayStatusResponse{Status: "deleted"}, nil
}
