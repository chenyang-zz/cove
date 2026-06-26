package response

import (
	"net/http"

	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data})
}

func FromError(c *gin.Context, err error) {
	status, code, message := xerr.ToHTTP(err)
	c.JSON(status, Envelope{Code: code, Message: message})
}

func BadRequest(c *gin.Context, err error) {
	FromError(c, xerr.Validation(err))
}
