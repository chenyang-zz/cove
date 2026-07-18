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

// ListPairingsLogic contains the listPairings use case.
type ListPairingsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListPairingsLogic creates a ListPairingsLogic.
func NewListPairingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPairingsLogic {
	return &ListPairingsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.listpairings"),
	}
}

// ListPairings 查询渠道账号的私聊配对请求
func (l *ListPairingsLogic) ListPairings(userID uuid.UUID, input *request.UriGatewayIDRequest) ([]*response.ChannelPairingResponse, error) {
	accountID, err := gatewayID(input)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.ChannelGatewayRepo.FindAccountByID(l.ctx, userID, accountID); err != nil {
		return nil, err
	}
	rows, err := l.svcCtx.ChannelGatewayRepo.ListPairings(l.ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ChannelPairingResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, pairingResponse(row))
	}
	return out, nil
}
