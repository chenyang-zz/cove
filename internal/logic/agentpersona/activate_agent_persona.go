package agentpersona

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

// ActivateAgentPersonaLogic contains the activateAgentPersona use case.
type ActivateAgentPersonaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewActivateAgentPersonaLogic creates a ActivateAgentPersonaLogic.
func NewActivateAgentPersonaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ActivateAgentPersonaLogic {
	return &ActivateAgentPersonaLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.agentpersona.activateagentpersona"),
	}
}

// ActivateAgentPersona 激活智能体角色
func (l *ActivateAgentPersonaLogic) ActivateAgentPersona(userID uuid.UUID, input *request.UriAgentPersonaIDRequest) (*response.AgentPersonaResponse, error) {
	personaID, err := personIDFromInput(input)
	if err != nil {
		return nil, err
	}

	persona, err := l.svcCtx.AgentPersonaRepo.FindByID(l.ctx, userID, personaID)
	if err != nil {
		return nil, err
	}

	err = l.svcCtx.AgentPersonaRepo.ActivateByID(l.ctx, userID, persona.ID)
	if err != nil {
		return nil, err
	}
	persona, err = l.svcCtx.AgentPersonaRepo.FindByID(l.ctx, userID, personaID)
	if err != nil {
		return nil, err
	}

	avatarUrl := ""
	if persona.AvatarKey != "" {
		avatarUrl = l.svcCtx.URLSigner.URL(persona.AvatarKey)
	}

	return mapper.AgentPersonaToResponse(persona, avatarUrl), nil
}
