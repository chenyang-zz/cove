package handler

import (
	"github.com/boxify/api-go/internal/app"
	"github.com/boxify/api-go/internal/transport/http/middleware"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	service *app.AuthService
}

func NewAuthHandler(service *app.AuthService) AuthHandler {
	return AuthHandler{service: service}
}

func (h AuthHandler) Register(c *gin.Context) {
	var body request.RegisterRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err)
		return
	}
	out, err := h.service.Register(c.Request.Context(), app.RegisterInput{
		Username: body.Username,
		Nickname: body.Nickname,
		Email:    body.Email,
		Avatar:   body.Avatar,
		Password: body.Password,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AuthHandler) Login(c *gin.Context) {
	var body request.LoginRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err)
		return
	}
	out, err := h.service.Login(c.Request.Context(), app.LoginInput{
		Login: body.Login, Password: body.Password,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h AuthHandler) Refresh(c *gin.Context) {
	var body request.RefreshRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err)
		return
	}
	out, err := h.service.Refresh(c.Request.Context(), app.RefreshInput{
		RefreshToken: body.RefreshToken,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (AuthHandler) Me(c *gin.Context) {
	userID, _ := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	response.OK(c, map[string]string{"id": userID.String()})
}
