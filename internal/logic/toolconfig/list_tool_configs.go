package toolconfig

import (
	"context"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
	"log/slog"
)

// ListToolConfigsLogic contains the listToolConfigs use case.
type ListToolConfigsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListToolConfigsLogic creates a ListToolConfigsLogic.
func NewListToolConfigsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListToolConfigsLogic {
	return &ListToolConfigsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.toolconfig.listtoolconfigs"),
	}
}

// ListToolConfigs 查询工具配置列表
func (l *ListToolConfigsLogic) ListToolConfigs(userID uuid.UUID) (*response.ListResponse[*response.ToolConfigResponse], error) {
	_ = l
	return nil, nil
}
