package mcpserver

import (
	"context"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
	"log/slog"
)

// ToggleMCPServerLogic contains the toggleMCPServer use case.
type ToggleMCPServerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewToggleMCPServerLogic creates a ToggleMCPServerLogic.
func NewToggleMCPServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ToggleMCPServerLogic {
	return &ToggleMCPServerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.mcpserver.togglemcpserver"),
	}
}

// ToggleMCPServer 切换mcp状态
func (l *ToggleMCPServerLogic) ToggleMCPServer(userID uuid.UUID, input *request.ToggleMCPServerRequest) (*response.MCPServerResponse, error) {
	_ = l
	return nil, nil
}
