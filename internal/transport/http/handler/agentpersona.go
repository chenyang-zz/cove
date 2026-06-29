package handler

import (
	agentpersonalogic "github.com/boxify/api-go/internal/logic/agentpersona"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type AgentPersonaHandler struct {
	svc *svc.ServiceContext
}

func NewAgentPersonaHandler(svcCtx *svc.ServiceContext) AgentPersonaHandler {
	return AgentPersonaHandler{svc: svcCtx}
}

func (h AgentPersonaHandler) ListAgentPersonas(c *gin.Context) {
	var query request.ListAgentPersonasRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := agentpersonalogic.NewListAgentPersonasLogic(c.Request.Context(), h.svc).ListAgentPersonas(userID, &query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AgentPersonaHandler) CreateAgentPersona(c *gin.Context) {
	var body request.CreateAgentPersonaRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := agentpersonalogic.NewCreateAgentPersonaLogic(c.Request.Context(), h.svc).CreateAgentPersona(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AgentPersonaHandler) UpdateAgentPersona(c *gin.Context) {
	var body request.UpdateAgentPersonaRequest
	if err := c.ShouldBindUri(&body.UriAgentPersonaIDRequest); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := agentpersonalogic.NewUpdateAgentPersonaLogic(c.Request.Context(), h.svc).UpdateAgentPersona(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AgentPersonaHandler) ActivateAgentPersona(c *gin.Context) {
	var body request.UriAgentPersonaIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := agentpersonalogic.NewActivateAgentPersonaLogic(c.Request.Context(), h.svc).ActivateAgentPersona(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AgentPersonaHandler) DeleteAgentPersona(c *gin.Context) {
	var body request.UriAgentPersonaIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	if err := agentpersonalogic.NewDeleteAgentPersonaLogic(c.Request.Context(), h.svc).DeleteAgentPersona(userID, &body); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}
