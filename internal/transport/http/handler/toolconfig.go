package handler

import (
	toolconfiglogic "github.com/boxify/api-go/internal/logic/toolconfig"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type ToolConfigHandler struct {
	svc *svc.ServiceContext
}

func NewToolConfigHandler(svcCtx *svc.ServiceContext) ToolConfigHandler {
	return ToolConfigHandler{svc: svcCtx}
}

func (h ToolConfigHandler) ListToolConfigs(c *gin.Context) {
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := toolconfiglogic.NewListToolConfigsLogic(c.Request.Context(), h.svc).ListToolConfigs(userID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h ToolConfigHandler) ToggleTool(c *gin.Context) {
	var body request.ToggleToolRequest
	if err := c.ShouldBindUri(&body.UriToolKeyRequest); err != nil {
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
	if err := toolconfiglogic.NewToggleToolLogic(c.Request.Context(), h.svc).ToggleTool(userID, &body); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}
