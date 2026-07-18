package handler

import (
	"strings"

	gatewaylogic "github.com/boxify/api-go/internal/logic/gateway"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GatewayHandler 暴露网关控制面 API。
type GatewayHandler struct{ service *gatewaylogic.Service }

// NewGatewayHandler 创建网关处理器。
func NewGatewayHandler(svcCtx *svc.ServiceContext) GatewayHandler {
	return GatewayHandler{service: gatewaylogic.NewService(svcCtx)}
}

func (h GatewayHandler) Providers(c *gin.Context) { response.OK(c, h.service.Providers()) }

func (h GatewayHandler) ListAccounts(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		return
	}
	out, err := h.service.ListAccounts(c.Request.Context(), userID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) CreateAccount(c *gin.Context) {
	var input request.CreateChannelAccountRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, ok := gatewayUserID(c)
	if !ok {
		return
	}
	out, err := h.service.CreateAccount(c.Request.Context(), userID, &input)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) GetAccount(c *gin.Context) {
	userID, accountID, ok := gatewayUserAndParamID(c, "id")
	if !ok {
		return
	}
	out, err := h.service.GetAccount(c.Request.Context(), userID, accountID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) UpdateAccount(c *gin.Context) {
	userID, accountID, ok := gatewayUserAndParamID(c, "id")
	if !ok {
		return
	}
	var input request.UpdateChannelAccountRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	out, err := h.service.UpdateAccount(c.Request.Context(), userID, accountID, &input)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) DeleteAccount(c *gin.Context) {
	userID, accountID, ok := gatewayUserAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), userID, accountID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, response.GatewayStatusResponse{Status: "deleted"})
}

func (h GatewayHandler) TestAccount(c *gin.Context) {
	userID, accountID, ok := gatewayUserAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.service.TestAccount(c.Request.Context(), userID, accountID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"status": "healthy"})
}

func (h GatewayHandler) ListPairings(c *gin.Context) {
	userID, accountID, ok := gatewayUserAndParamID(c, "id")
	if !ok {
		return
	}
	out, err := h.service.ListPairings(c.Request.Context(), userID, accountID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) ApprovePairing(c *gin.Context) {
	userID, identityID, ok := gatewayUserAndParamID(c, "id")
	if !ok {
		return
	}
	out, err := h.service.ApprovePairing(c.Request.Context(), userID, identityID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) DenyPairing(c *gin.Context) {
	userID, identityID, ok := gatewayUserAndParamID(c, "id")
	if !ok {
		return
	}
	out, err := h.service.DenyPairing(c.Request.Context(), userID, identityID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) ListBindings(c *gin.Context) {
	userID, ok := gatewayUserID(c)
	if !ok {
		return
	}
	var accountID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("account_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			response.FromError(c, xerr.BadRequest("渠道账号 ID 无效"))
			return
		}
		accountID = &parsed
	}
	out, err := h.service.ListBindings(c.Request.Context(), userID, accountID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) CreateBinding(c *gin.Context) {
	var input request.CreateChannelBindingRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, ok := gatewayUserID(c)
	if !ok {
		return
	}
	out, err := h.service.CreateBinding(c.Request.Context(), userID, &input)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) GetBinding(c *gin.Context) {
	userID, bindingID, ok := gatewayUserAndParamID(c, "id")
	if !ok {
		return
	}
	out, err := h.service.GetBinding(c.Request.Context(), userID, bindingID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) UpdateBinding(c *gin.Context) {
	userID, bindingID, ok := gatewayUserAndParamID(c, "id")
	if !ok {
		return
	}
	var input request.UpdateChannelBindingRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	out, err := h.service.UpdateBinding(c.Request.Context(), userID, bindingID, &input)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) DeleteBinding(c *gin.Context) {
	userID, bindingID, ok := gatewayUserAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteBinding(c.Request.Context(), userID, bindingID); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, response.GatewayStatusResponse{Status: "deleted"})
}

func gatewayUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return uuid.Nil, false
	}
	return userID, true
}

func gatewayUserAndParamID(c *gin.Context, name string) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := gatewayUserID(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		response.FromError(c, xerr.BadRequest("ID 无效"))
		return uuid.Nil, uuid.Nil, false
	}
	return userID, id, true
}
