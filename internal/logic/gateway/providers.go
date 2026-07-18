package gateway

import (
	"context"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
	"log/slog"
)

// ProvidersLogic contains the providers use case.
type ProvidersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewProvidersLogic creates a ProvidersLogic.
func NewProvidersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProvidersLogic {
	return &ProvidersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.providers"),
	}
}

// Providers 查询网关支持的 Provider、配置字段和能力矩阵
func (l *ProvidersLogic) Providers(userID uuid.UUID) ([]response.GatewayProviderResponse, error) {
	_ = userID
	if l.svcCtx == nil || l.svcCtx.ChannelRegistry == nil {
		return nil, nil
	}
	descriptors := l.svcCtx.ChannelRegistry.Descriptors()
	out := make([]response.GatewayProviderResponse, 0, len(descriptors))
	for _, descriptor := range descriptors {
		out = append(out, providerResponse(descriptor))
	}
	return out, nil
}
