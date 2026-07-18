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

// GetAccountLogic contains the getAccount use case.
type GetAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewGetAccountLogic creates a GetAccountLogic.
func NewGetAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountLogic {
	return &GetAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.getaccount"),
	}
}

// GetAccount 查询渠道账号详情
func (l *GetAccountLogic) GetAccount(userID uuid.UUID, input *request.UriGatewayIDRequest) (*response.ChannelAccountResponse, error) {
	accountID, err := gatewayID(input)
	if err != nil {
		return nil, err
	}
	row, err := l.svcCtx.ChannelGatewayRepo.FindAccountByID(l.ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	return accountResponse(l.svcCtx, row)
}
