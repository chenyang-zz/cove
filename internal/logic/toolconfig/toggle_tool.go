package toolconfig

import (
	"context"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
	"log/slog"
)

// ToggleToolLogic contains the toggleTool use case.
type ToggleToolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewToggleToolLogic creates a ToggleToolLogic.
func NewToggleToolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ToggleToolLogic {
	return &ToggleToolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.toolconfig.toggletool"),
	}
}

// ToggleTool 开启/关闭工具
func (l *ToggleToolLogic) ToggleTool(userID uuid.UUID, input *request.ToggleToolRequest) error {
	_ = l
	return nil
}
