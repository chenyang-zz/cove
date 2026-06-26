package handler

import (
	"github.com/boxify/api-go/internal/app"
	"github.com/boxify/api-go/internal/transport/http/middleware"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ModelConfigHandler struct {
	service *app.ModelConfigService
}

func NewModelConfigHandler(service *app.ModelConfigService) ModelConfigHandler {
	return ModelConfigHandler{service: service}
}

func (h ModelConfigHandler) Create(c *gin.Context) {
	var body request.CreateModelConfigRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err)
		return
	}
	userID, _ := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	row, err := h.service.Create(c.Request.Context(), app.CreateModelConfigInput{
		UserID:    userID,
		Type:      body.Type,
		Provider:  body.Provider,
		Model:     body.Model,
		BaseURL:   body.BaseURL,
		APIKey:    body.APIKey,
		IsDefault: body.IsDefault,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, row)
}

func (h ModelConfigHandler) List(c *gin.Context) {
	userID, _ := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	rows, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, rows)
}
