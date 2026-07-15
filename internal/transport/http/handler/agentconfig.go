package handler

import (
	agentconfiglogic "github.com/boxify/api-go/internal/logic/agentconfig"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type AgentConfigHandler struct {
	svc *svc.ServiceContext
}

func NewAgentConfigHandler(svcCtx *svc.ServiceContext) AgentConfigHandler {
	return AgentConfigHandler{svc: svcCtx}
}

func (h AgentConfigHandler) ListAgentConfigs(c *gin.Context) {
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := agentconfiglogic.NewListAgentConfigsLogic(c.Request.Context(), h.svc).ListAgentConfigs(userID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AgentConfigHandler) CreateAgentConfig(c *gin.Context) {
	var body request.CreateAgentConfigRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := agentconfiglogic.NewCreateAgentConfigLogic(c.Request.Context(), h.svc).CreateAgentConfig(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AgentConfigHandler) GetAgentConfig(c *gin.Context) {
	var input request.UriAgentConfigIDRequest
	if err := c.ShouldBindUri(&input); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := agentconfiglogic.NewGetAgentConfigLogic(c.Request.Context(), h.svc).GetAgentConfig(userID, &input)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AgentConfigHandler) UpdateAgentConfig(c *gin.Context) {
	var body request.UpdateAgentConfigRequest
	if err := c.ShouldBindUri(&body.UriAgentConfigIDRequest); err != nil {
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
	out, err := agentconfiglogic.NewUpdateAgentConfigLogic(c.Request.Context(), h.svc).UpdateAgentConfig(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AgentConfigHandler) DeleteAgentConfig(c *gin.Context) {
	var input request.UriAgentConfigIDRequest
	if err := c.ShouldBindUri(&input); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	if err := agentconfiglogic.NewDeleteAgentConfigLogic(c.Request.Context(), h.svc).DeleteAgentConfig(userID, &input); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h AgentConfigHandler) SetDefaultAgentConfig(c *gin.Context) {
	var input request.UriAgentConfigIDRequest
	if err := c.ShouldBindUri(&input); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := agentconfiglogic.NewSetDefaultAgentConfigLogic(c.Request.Context(), h.svc).SetDefaultAgentConfig(userID, &input)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AgentConfigHandler) OptimizePrompt(c *gin.Context) {
	var body request.OptimizePromptRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := agentconfiglogic.NewOptimizePromptLogic(c.Request.Context(), h.svc).OptimizePrompt(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}
