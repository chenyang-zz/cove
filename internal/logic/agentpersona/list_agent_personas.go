package agentpersona

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

// ListAgentPersonasLogic contains the listAgentPersonas use case.
type ListAgentPersonasLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListAgentPersonasLogic creates a ListAgentPersonasLogic.
func NewListAgentPersonasLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAgentPersonasLogic {
	return &ListAgentPersonasLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.agentpersona.listagentpersonas"),
	}
}

// ListAgentPersonas 查询智能体角色列表
func (l *ListAgentPersonasLogic) ListAgentPersonas(userID uuid.UUID, input *request.ListAgentPersonasRequest) (*response.ListResponse[*response.AgentPersonaResponse], error) {
	personas, err := l.svcCtx.AgentPersonaRepo.List(l.ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(personas) == 0 {
		// 创建默认角色
		defaultPersona := models.DefaultPersona
		_, err = l.svcCtx.AgentPersonaRepo.Create(l.ctx, userID, &defaultPersona)
		if err != nil {
			return nil, err
		}
		personas, err = l.svcCtx.AgentPersonaRepo.List(l.ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	filterPersonas := make([]*models.AgentPersona, 0, len(personas))
	// 默认只返回「单个角色」（隐藏仅作为卡组成员的角色，保持列表干净）
	if !input.All {
		for _, persona := range personas {
			if !persona.InGroupOnly {
				filterPersonas = append(filterPersonas, persona)
			}
		}
	} else {
		filterPersonas = personas
	}

	resList := make([]*response.AgentPersonaResponse, 0, len(filterPersonas))
	for _, persona := range filterPersonas {
		avatarUrl := ""
		if persona.AvatarKey != "" {
			avatarUrl = l.svcCtx.URLSigner.URL(persona.AvatarKey)
		}
		resList = append(resList, mapper.AgentPersonaToResponse(persona, avatarUrl))
	}

	return &response.ListResponse[*response.AgentPersonaResponse]{
		List: resList,
	}, nil
}
