package gateway

import (
	"context"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"log/slog"
)

// TestAccountLogic contains the testAccount use case.
type TestAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewTestAccountLogic creates a TestAccountLogic.
func NewTestAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TestAccountLogic {
	return &TestAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.testaccount"),
	}
}

// TestAccount 验证渠道账号凭据和平台连通性
func (l *TestAccountLogic) TestAccount(userID uuid.UUID, input *request.UriGatewayIDRequest) (*response.GatewayStatusResponse, error) {
	accountID, err := gatewayID(input)
	if err != nil {
		return nil, err
	}
	row, err := l.svcCtx.ChannelGatewayRepo.FindAccountByID(l.ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	provider, ok := l.svcCtx.ChannelRegistry.Get(corechannel.ProviderName(row.Provider))
	if !ok {
		return nil, xerr.BadRequest("渠道 Provider 不可用")
	}
	tester, ok := provider.(corechannel.Tester)
	if !ok {
		return &response.GatewayStatusResponse{Status: "healthy"}, nil
	}
	account, err := accountConfig(l.svcCtx, row)
	if err != nil {
		return nil, err
	}
	testCtx, cancel := context.WithTimeout(l.ctx, 15*time.Second)
	defer cancel()
	if err := tester.TestAccount(testCtx, account); err != nil {
		_ = l.svcCtx.ChannelGatewayRepo.UpdateAccountHealth(l.ctx, accountID, models.ChannelAccountStatusDegraded, "连接测试失败")
		return nil, xerr.BadRequest("渠道账号连接测试失败")
	}
	if err := l.svcCtx.ChannelGatewayRepo.UpdateAccountHealth(l.ctx, accountID, models.ChannelAccountStatusHealthy, ""); err != nil {
		return nil, err
	}
	return &response.GatewayStatusResponse{Status: "healthy"}, nil
}
