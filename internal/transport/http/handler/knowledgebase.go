package handler

import (
	knowledgebaselogic "github.com/boxify/api-go/internal/logic/knowledgebase"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type KnowledgeBaseHandler struct {
	svc *svc.ServiceContext
}

func NewKnowledgeBaseHandler(svcCtx *svc.ServiceContext) KnowledgeBaseHandler {
	return KnowledgeBaseHandler{svc: svcCtx}
}

func (h KnowledgeBaseHandler) GetKnowledgeBase(c *gin.Context) {
	var query request.UriKnowledgeBaseIDRequest
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
	out, err := knowledgebaselogic.NewGetKnowledgeBaseLogic(c.Request.Context(), h.svc).GetKnowledgeBase(userID, &query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h KnowledgeBaseHandler) GetKnowledgeBaseList(c *gin.Context) {
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := knowledgebaselogic.NewGetKnowledgeBaseListLogic(c.Request.Context(), h.svc).GetKnowledgeBaseList(userID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h KnowledgeBaseHandler) CreateKnowledgeBase(c *gin.Context) {
	var body request.CreateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := knowledgebaselogic.NewCreateKnowledgeBaseLogic(c.Request.Context(), h.svc).CreateKnowledgeBase(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h KnowledgeBaseHandler) UpdateKnowledgeBase(c *gin.Context) {
	var body request.UpdateKnowledgeBaseRequest
	if err := c.ShouldBindUri(&body.UriKnowledgeBaseIDRequest); err != nil {
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
	out, err := knowledgebaselogic.NewUpdateKnowledgeBaseLogic(c.Request.Context(), h.svc).UpdateKnowledgeBase(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h KnowledgeBaseHandler) DeleteKnowledgeBase(c *gin.Context) {
	var body request.UriKnowledgeBaseIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	if err := knowledgebaselogic.NewDeleteKnowledgeBaseLogic(c.Request.Context(), h.svc).DeleteKnowledgeBase(userID, &body); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h KnowledgeBaseHandler) EnabledChat(c *gin.Context) {
	var body request.EnabledChatRequest
	if err := c.ShouldBindUri(&body.UriKnowledgeBaseIDRequest); err != nil {
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
	out, err := knowledgebaselogic.NewEnabledChatLogic(c.Request.Context(), h.svc).EnabledChat(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

// SetDefaultKnowledgeBase 将指定知识库设置为当前用户的默认知识库。
func (h KnowledgeBaseHandler) SetDefaultKnowledgeBase(c *gin.Context) {
	var input request.UriKnowledgeBaseIDRequest
	if err := c.ShouldBindUri(&input); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := knowledgebaselogic.NewSetDefaultKnowledgeBaseLogic(c.Request.Context(), h.svc).SetDefaultKnowledgeBase(userID, &input)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}
