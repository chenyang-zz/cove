package gateway

import (
	"context"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
	"log/slog"
)

// ListAccountsLogic contains the listAccounts use case.
type ListAccountsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListAccountsLogic creates a ListAccountsLogic.
func NewListAccountsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAccountsLogic {
	return &ListAccountsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.listaccounts"),
	}
}

// ListAccounts 查询当前用户的渠道账号，凭据仅返回掩码
func (l *ListAccountsLogic) ListAccounts(userID uuid.UUID) ([]*response.ChannelAccountResponse, error) {
	rows, err := l.svcCtx.ChannelGatewayRepo.ListAccounts(l.ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ChannelAccountResponse, 0, len(rows))
	for _, row := range rows {
		item, mapErr := accountResponse(l.svcCtx, row)
		if mapErr != nil {
			return nil, mapErr
		}
		out = append(out, item)
	}
	return out, nil
}
