package handler

import (
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func (HealthHandler) Hello(c *gin.Context) {
	response.OK(c, map[string]string{"message": "welcome to api-go"})
}

func (HealthHandler) Health(c *gin.Context) {
	response.OK(c, map[string]string{"status": "ok"})
}
