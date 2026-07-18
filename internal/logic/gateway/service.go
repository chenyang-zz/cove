// Package gateway 实现消息网关控制面用例和数据面编排。
package gateway

import (
	"log/slog"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
)

// Service 承载网关数据面的入站、媒体和账号运行配置能力。
type Service struct {
	svc *svc.ServiceContext
	log *slog.Logger
}

// NewService 创建网关数据面服务。
func NewService(svcCtx *svc.ServiceContext) *Service {
	return &Service{svc: svcCtx, log: xlog.Component("logic.gateway")}
}

// AccountConfig 解密 Provider 运行所需的账号快照。
func (s *Service) AccountConfig(row *models.ChannelAccount) (corechannel.AccountConfig, error) {
	return accountConfig(s.svc, row)
}
