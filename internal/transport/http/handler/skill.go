package handler

import (
	skilllogic "github.com/boxify/api-go/internal/logic/skill"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type SkillHandler struct {
	svc *svc.ServiceContext
}

func NewSkillHandler(svcCtx *svc.ServiceContext) SkillHandler {
	return SkillHandler{svc: svcCtx}
}

func (h SkillHandler) ListSkills(c *gin.Context) {
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := skilllogic.NewListSkillsLogic(c.Request.Context(), h.svc).ListSkills(userID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h SkillHandler) ListBuiltinSkills(c *gin.Context) {
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := skilllogic.NewListBuiltinSkillsLogic(c.Request.Context(), h.svc).ListBuiltinSkills(userID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h SkillHandler) CreateSkill(c *gin.Context) {
	var body request.CreateSkillRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := skilllogic.NewCreateSkillLogic(c.Request.Context(), h.svc).CreateSkill(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h SkillHandler) CopyBuiltinSkill(c *gin.Context) {
	var body request.UriSkillIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := skilllogic.NewCopyBuiltinSkillLogic(c.Request.Context(), h.svc).CopyBuiltinSkill(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h SkillHandler) OptimizeSkillPrompt(c *gin.Context) {
	var body request.OptimizeSkillPromptRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := skilllogic.NewOptimizeSkillPromptLogic(c.Request.Context(), h.svc).OptimizeSkillPrompt(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h SkillHandler) UpdateSkill(c *gin.Context) {
	var body request.UpdateSkillRequest
	if err := c.ShouldBindUri(&body.UriSkillIDRequest); err != nil {
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
	out, err := skilllogic.NewUpdateSkillLogic(c.Request.Context(), h.svc).UpdateSkill(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h SkillHandler) DeleteSkill(c *gin.Context) {
	var body request.UriSkillIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	if err := skilllogic.NewDeleteSkillLogic(c.Request.Context(), h.svc).DeleteSkill(userID, &body); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}
