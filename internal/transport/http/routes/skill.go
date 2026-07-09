/**
 * @Time   : 2026/7/9 18:55
 * @Author : chenyangzhao542@gmail.com
 * @File   : skill.go
 **/

package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterSkillRoutes(api *gin.RouterGroup, skill handler.SkillHandler, authMiddleware gin.HandlerFunc) {
	skillRoutes := api.Group("/skill", authMiddleware)

	// @auth(user_id)
	// @description 查询skill列表
	// @output response.ListResponse[*response.SkillResponse]
	skillRoutes.GET("/", skill.ListSkills)
	skillRoutes.GET("/list", skill.ListSkills)

	// @auth(user_id)
	// @description 查询内置skill
	// @output response.ListResponse[*response.SkillResponse]
	skillRoutes.GET("/builtin", skill.ListBuiltinSkills)

	// @auth(user_id)
	// @description 创建skill
	// @input request.CreateSkillRequest
	// @output response.SkillResponse
	skillRoutes.POST("", skill.CreateSkill)
	skillRoutes.POST("/create", skill.CreateSkill)

	// @auth(user_id)
	// @description 把内置技能复制为用户技能
	// @input request.UriSkillIDRequest
	// @output response.SkillResponse
	skillRoutes.POST("/builtin/:skill_id", skill.CopyBuiltinSkill)

	// @auth(user_id)
	// @description 优化提示词
	// @input request.OptimizeSkillPromptRequest
	skillRoutes.POST("/optimize-prompt", skill.OptimizeSkillPrompt)

	// @auth(user_id)
	// @description 更新skill
	// @input request.UpdateSkillRequest
	// @output response.SkillResponse
	skillRoutes.PATCH("/:skill_id", skill.UpdateSkill)
	skillRoutes.POST("/:skill_id/update", skill.UpdateSkill)

	// @auth(user_id)
	// @description 删除skill
	// @input request.UriSkillIDRequest
	skillRoutes.DELETE("/:skill_id", skill.DeleteSkill)
	skillRoutes.POST("/:skill_id/delete", skill.DeleteSkill)
}
