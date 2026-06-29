package agentpersona

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
)

// DeleteAgentPersonaLogic contains the deleteAgentPersona use case.
type DeleteAgentPersonaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDeleteAgentPersonaLogic creates a DeleteAgentPersonaLogic.
func NewDeleteAgentPersonaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAgentPersonaLogic {
	return &DeleteAgentPersonaLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.agentpersona.deleteagentpersona"),
	}
}

// DeleteAgentPersona 删除智能体角色
func (l *DeleteAgentPersonaLogic) DeleteAgentPersona(userID uuid.UUID, input *request.UriAgentPersonaIDRequest) error {
	personaID, err := personIDFromInput(input)
	if err != nil {
		return err
	}

	err = l.svcCtx.AgentPersonaRepo.Delete(l.ctx, userID, personaID)
	if err != nil {
		return err
	}

	return nil
}
