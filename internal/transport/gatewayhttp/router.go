// Package gatewayhttp 暴露独立网关进程的公共数据面。
package gatewayhttp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/infrastructure/channel/webhook"
	gatewaylogic "github.com/boxify/api-go/internal/logic/gateway"
	"github.com/boxify/api-go/internal/svc"
	"github.com/gin-gonic/gin"
)

// WebhookInboundRequest 是通用 Webhook 的稳定入站协议。
type WebhookInboundRequest struct {
	EventID   string `json:"event_id" binding:"required,max=255"`
	ChatType  string `json:"chat_type" binding:"required,oneof=direct group"`
	ChatID    string `json:"chat_id" binding:"required,max=255"`
	ThreadID  string `json:"thread_id" binding:"omitempty,max=255"`
	MessageID string `json:"message_id" binding:"omitempty,max=255"`
	Sender    struct {
		ID          string `json:"id" binding:"required,max=255"`
		DisplayName string `json:"display_name" binding:"omitempty,max=255"`
		Username    string `json:"username" binding:"omitempty,max=255"`
		IsBot       bool   `json:"is_bot"`
	} `json:"sender" binding:"required"`
	Text       string                       `json:"text"`
	Reply      *corechannel.ReplyReference  `json:"reply"`
	Mentioned  bool                         `json:"mentioned"`
	Media      []corechannel.MediaReference `json:"media"`
	OccurredAt *time.Time                   `json:"occurred_at"`
}

// NewRouter 构造健康检查和通用 Webhook 路由。
func NewRouter(svcCtx *svc.ServiceContext) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	service := gatewaylogic.NewService(svcCtx)
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.POST("/gateway/v1/hooks/:public_id", webhookHandler(svcCtx, service))
	return router
}

func webhookHandler(svcCtx *svc.ServiceContext, service *gatewaylogic.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		maxBytes := svcCtx.Config.Gateway.MaxRequestBytes
		if maxBytes <= 0 {
			maxBytes = 1 << 20
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			publicError(c, http.StatusRequestEntityTooLarge, "request body is too large")
			return
		}
		account, err := svcCtx.ChannelGatewayRepo.FindAccountByPublicID(c.Request.Context(), c.Param("public_id"))
		if err != nil || account.Provider != string(corechannel.ProviderWebhook) {
			publicError(c, http.StatusNotFound, "webhook not found")
			return
		}
		accountConfig, err := service.AccountConfig(account)
		if err != nil {
			publicError(c, http.StatusInternalServerError, "webhook is unavailable")
			return
		}
		window, err := time.ParseDuration(svcCtx.Config.Gateway.WebhookSigningWindow)
		if err != nil || window <= 0 {
			window = 5 * time.Minute
		}
		if err := webhook.Verify(
			accountConfig.Credentials["signing_secret"],
			c.GetHeader("X-Cove-Timestamp"), c.GetHeader("X-Cove-Signature"), body, time.Now(), window,
		); err != nil {
			publicError(c, http.StatusUnauthorized, "invalid webhook signature")
			return
		}
		var input WebhookInboundRequest
		decoder := json.NewDecoder(strings.NewReader(string(body)))
		if err := decoder.Decode(&input); err != nil {
			publicError(c, http.StatusBadRequest, "invalid webhook payload")
			return
		}
		if strings.TrimSpace(input.EventID) == "" || strings.TrimSpace(input.ChatID) == "" || strings.TrimSpace(input.Sender.ID) == "" ||
			(input.ChatType != string(corechannel.ChatTypeDirect) && input.ChatType != string(corechannel.ChatTypeGroup)) {
			publicError(c, http.StatusBadRequest, "invalid webhook payload")
			return
		}
		occurredAt := time.Now()
		if input.OccurredAt != nil {
			occurredAt = *input.OccurredAt
		}
		event := corechannel.InboundEvent{
			ID: input.EventID, Provider: corechannel.ProviderWebhook, ProviderEventID: input.EventID,
			Route:             corechannel.Route{AccountID: account.ID.String(), ChatType: corechannel.ChatType(input.ChatType), ChatID: input.ChatID, ThreadID: input.ThreadID},
			Sender:            corechannel.SenderIdentity{ID: input.Sender.ID, DisplayName: input.Sender.DisplayName, Username: input.Sender.Username, IsBot: input.Sender.IsBot},
			PlatformMessageID: input.MessageID, Text: input.Text, Reply: input.Reply, Mentioned: input.Mentioned,
			Media: input.Media, OccurredAt: occurredAt, ReceivedAt: time.Now(),
		}
		inbox, created, err := service.HandleInbound(c.Request.Context(), account, event)
		if err != nil {
			if c.Request.Context().Err() != nil {
				publicError(c, http.StatusServiceUnavailable, "gateway unavailable")
			} else {
				publicError(c, http.StatusUnprocessableEntity, "event rejected")
			}
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"event_id": inbox.ID.String(), "duplicate": !created, "status": inbox.Status})
	}
}

func publicError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
