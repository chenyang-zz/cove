package handler

import (
	"encoding/json"
	"fmt"

	"github.com/boxify/api-go/internal/app"
	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/transport/http/middleware"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	service *app.ChatService
}

func NewChatHandler(service *app.ChatService) ChatHandler {
	return ChatHandler{service: service}
}

func (h ChatHandler) Stream(c *gin.Context) {
	var body request.ChatStreamRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err)
		return
	}
	userID, _ := c.MustGet(middleware.UserIDKey).(uuid.UUID)
	events, err := h.service.Stream(c.Request.Context(), domain.ChatStreamInput{
		UserID:  userID,
		Message: body.Message,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(200)

	for event := range events {
		name := event.Type
		payload := map[string]any{}
		switch event.Type {
		case "meta":
			payload["conversation_id"] = event.Text
			if title, ok := event.Stats["title"]; ok {
				payload["title"] = title
			}
		case "token":
			payload["text"] = event.Text
		case "done":
			payload["conversation_id"] = event.Text
		default:
			payload["text"] = event.Text
		}
		writeSSE(c.Writer, name, payload)
		c.Writer.Flush()
	}
}

func writeSSE(w gin.ResponseWriter, event string, data any) {
	encoded, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, encoded)
}
