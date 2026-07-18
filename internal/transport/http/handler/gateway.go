package handler

import (
	gatewaylogic "github.com/boxify/api-go/internal/logic/gateway"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type GatewayHandler struct {
	svc *svc.ServiceContext
}

func NewGatewayHandler(svcCtx *svc.ServiceContext) GatewayHandler {
	return GatewayHandler{svc: svcCtx}
}

func (h GatewayHandler) Providers(c *gin.Context) {
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewProvidersLogic(c.Request.Context(), h.svc).Providers(userID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) ListAccounts(c *gin.Context) {
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewListAccountsLogic(c.Request.Context(), h.svc).ListAccounts(userID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) CreateAccount(c *gin.Context) {
	var body request.CreateChannelAccountRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewCreateAccountLogic(c.Request.Context(), h.svc).CreateAccount(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) GetAccount(c *gin.Context) {
	var query request.UriGatewayIDRequest
	if err := c.ShouldBindUri(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewGetAccountLogic(c.Request.Context(), h.svc).GetAccount(userID, &query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) UpdateAccount(c *gin.Context) {
	var body request.UpdateChannelAccountDocRequest
	if err := c.ShouldBindUri(&body.UriGatewayIDRequest); err != nil {
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
	out, err := gatewaylogic.NewUpdateAccountLogic(c.Request.Context(), h.svc).UpdateAccount(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) DeleteAccount(c *gin.Context) {
	var body request.UriGatewayIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewDeleteAccountLogic(c.Request.Context(), h.svc).DeleteAccount(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) TestAccount(c *gin.Context) {
	var body request.UriGatewayIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewTestAccountLogic(c.Request.Context(), h.svc).TestAccount(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) ListPairings(c *gin.Context) {
	var query request.UriGatewayIDRequest
	if err := c.ShouldBindUri(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewListPairingsLogic(c.Request.Context(), h.svc).ListPairings(userID, &query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) ApprovePairing(c *gin.Context) {
	var body request.UriGatewayIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewApprovePairingLogic(c.Request.Context(), h.svc).ApprovePairing(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) DenyPairing(c *gin.Context) {
	var body request.UriGatewayIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewDenyPairingLogic(c.Request.Context(), h.svc).DenyPairing(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) ListBindings(c *gin.Context) {
	var query request.ListChannelBindingsRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewListBindingsLogic(c.Request.Context(), h.svc).ListBindings(userID, &query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) CreateBinding(c *gin.Context) {
	var body request.CreateChannelBindingRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewCreateBindingLogic(c.Request.Context(), h.svc).CreateBinding(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) GetBinding(c *gin.Context) {
	var query request.UriGatewayIDRequest
	if err := c.ShouldBindUri(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewGetBindingLogic(c.Request.Context(), h.svc).GetBinding(userID, &query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) UpdateBinding(c *gin.Context) {
	var body request.UpdateChannelBindingDocRequest
	if err := c.ShouldBindUri(&body.UriGatewayIDRequest); err != nil {
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
	out, err := gatewaylogic.NewUpdateBindingLogic(c.Request.Context(), h.svc).UpdateBinding(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h GatewayHandler) DeleteBinding(c *gin.Context) {
	var body request.UriGatewayIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := gatewaylogic.NewDeleteBindingLogic(c.Request.Context(), h.svc).DeleteBinding(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}
