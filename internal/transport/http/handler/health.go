package handler

import (
	healthlogic "github.com/boxify/api-go/internal/logic/health"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	svc *svc.ServiceContext
}

func NewHealthHandler(svcCtx *svc.ServiceContext) HealthHandler {
	return HealthHandler{svc: svcCtx}
}

func (h HealthHandler) Health(c *gin.Context) {
	out, err := healthlogic.NewHealthLogic(c.Request.Context(), h.svc).Health()
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}
