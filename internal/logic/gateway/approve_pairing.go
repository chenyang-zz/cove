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

// ApprovePairingLogic contains the approvePairing use case.
type ApprovePairingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewApprovePairingLogic creates a ApprovePairingLogic.
func NewApprovePairingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApprovePairingLogic {
	return &ApprovePairingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.approvepairing"),
	}
}

// ApprovePairing 批准私聊配对请求
func (l *ApprovePairingLogic) ApprovePairing(userID uuid.UUID, input *request.UriGatewayIDRequest) (*response.ChannelPairingResponse, error) {
	identityID, err := gatewayID(input)
	if err != nil {
		return nil, err
	}
	return decidePairing(l.ctx, l.svcCtx, userID, identityID, true)
}
