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

// DenyPairingLogic contains the denyPairing use case.
type DenyPairingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDenyPairingLogic creates a DenyPairingLogic.
func NewDenyPairingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DenyPairingLogic {
	return &DenyPairingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.denypairing"),
	}
}

// DenyPairing 拒绝并阻止私聊配对请求
func (l *DenyPairingLogic) DenyPairing(userID uuid.UUID, input *request.UriGatewayIDRequest) (*response.ChannelPairingResponse, error) {
	identityID, err := gatewayID(input)
	if err != nil {
		return nil, err
	}
	return decidePairing(l.ctx, l.svcCtx, userID, identityID, false)
}
