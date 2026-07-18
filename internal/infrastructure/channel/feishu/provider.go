// Package feishu 实现飞书官方 WebSocket 长连接适配器。
package feishu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

// Provider 使用飞书官方 SDK 处理长连接、消息发送与媒体下载。
type Provider struct{}

// New 创建飞书 Provider。
func New() *Provider { return &Provider{} }

// Descriptor 返回飞书凭据表单和能力矩阵。
func (*Provider) Descriptor() corechannel.ProviderDescriptor {
	return corechannel.ProviderDescriptor{
		Name: corechannel.ProviderFeishu, DisplayName: "飞书",
		Description: "通过飞书开放平台 WebSocket 长连接接入私聊、群聊和话题",
		CredentialFields: []corechannel.FieldDescriptor{
			{Key: "app_id", Label: "App ID", Type: "text", Required: true},
			{Key: "app_secret", Label: "App Secret", Type: "password", Required: true, Sensitive: true},
			{Key: "verification_token", Label: "Verification Token", Type: "password", Sensitive: true},
			{Key: "encrypt_key", Label: "Encrypt Key", Type: "password", Sensitive: true},
		},
		SettingFields: []corechannel.FieldDescriptor{
			{Key: "bot_open_id", Label: "Bot Open ID", Type: "text", Description: "用于精确识别群聊提及"},
		},
		Capabilities: corechannel.Capabilities{
			DirectMessages: true, GroupMessages: true, Threads: true, Replies: true,
			Mentions: true, InboundImages: true, InboundFiles: true, OutboundText: true,
		},
		MaxTextLength: 30000,
	}
}

func credentials(account corechannel.AccountConfig) (string, string, error) {
	appID := strings.TrimSpace(account.Credentials["app_id"])
	appSecret := strings.TrimSpace(account.Credentials["app_secret"])
	if appID == "" || appSecret == "" {
		return "", "", errors.New("feishu app_id and app_secret are required")
	}
	return appID, appSecret, nil
}

// TestAccount 获取 tenant access token 以验证应用凭据。
func (*Provider) TestAccount(ctx context.Context, account corechannel.AccountConfig) error {
	appID, appSecret, err := credentials(account)
	if err != nil {
		return err
	}
	client := lark.NewClient(appID, appSecret)
	resp, err := client.GetTenantAccessTokenBySelfBuiltApp(ctx, &larkcore.SelfBuiltTenantAccessTokenReq{AppID: appID, AppSecret: appSecret})
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("feishu credential test failed: %s", resp.Msg)
	}
	return nil
}

// Receive 启动飞书官方 WebSocket 客户端并注册消息事件。
func (*Provider) Receive(ctx context.Context, account corechannel.AccountConfig, handler corechannel.EventHandler) error {
	appID, appSecret, err := credentials(account)
	if err != nil {
		return err
	}
	verificationToken := account.Credentials["verification_token"]
	encryptKey := account.Credentials["encrypt_key"]
	dispatcher := larkdispatcher.NewEventDispatcher(verificationToken, encryptKey)
	dispatcher.OnP2MessageReceiveV1(func(eventCtx context.Context, raw *larkim.P2MessageReceiveV1) error {
		event, ok := normalize(account, raw)
		if !ok || handler == nil {
			return nil
		}
		return handler.HandleInbound(eventCtx, event)
	})
	client := larkws.NewClient(appID, appSecret, larkws.WithEventHandler(dispatcher))
	return client.Start(ctx)
}

func normalize(account corechannel.AccountConfig, raw *larkim.P2MessageReceiveV1) (corechannel.InboundEvent, bool) {
	if raw == nil || raw.EventV2Base == nil || raw.EventV2Base.Header == nil || raw.Event == nil || raw.Event.Message == nil || raw.Event.Sender == nil {
		return corechannel.InboundEvent{}, false
	}
	message := raw.Event.Message
	senderID := ""
	if raw.Event.Sender.SenderId != nil {
		senderID = firstNonEmpty(stringValue(raw.Event.Sender.SenderId.OpenId), stringValue(raw.Event.Sender.SenderId.UserId), stringValue(raw.Event.Sender.SenderId.UnionId))
	}
	if senderID == "" {
		return corechannel.InboundEvent{}, false
	}
	chatType := corechannel.ChatTypeGroup
	if stringValue(message.ChatType) == "p2p" {
		chatType = corechannel.ChatTypeDirect
	}
	content := make(map[string]any)
	_ = json.Unmarshal([]byte(stringValue(message.Content)), &content)
	messageType := stringValue(message.MessageType)
	text, _ := content["text"].(string)
	media := make([]corechannel.MediaReference, 0, 1)
	messageID := stringValue(message.MessageId)
	switch messageType {
	case "image":
		if key, _ := content["image_key"].(string); key != "" {
			media = append(media, corechannel.MediaReference{ID: key, Kind: "image", MIMEType: "image/jpeg", URL: "feishu://" + messageID + "/image"})
		}
	case "file":
		if key, _ := content["file_key"].(string); key != "" {
			name, _ := content["file_name"].(string)
			media = append(media, corechannel.MediaReference{ID: key, Kind: "document", FileName: name, URL: "feishu://" + messageID + "/file"})
		}
	}
	mentioned := chatType == corechannel.ChatTypeDirect || mentionedBot(message.Mentions, account.Settings)
	occurredAt := time.Now()
	if millis, parseErr := strconv.ParseInt(stringValue(message.CreateTime), 10, 64); parseErr == nil {
		occurredAt = time.UnixMilli(millis)
	}
	var reply *corechannel.ReplyReference
	if parentID := stringValue(message.ParentId); parentID != "" {
		reply = &corechannel.ReplyReference{MessageID: parentID}
	}
	return corechannel.InboundEvent{
		ID: raw.EventV2Base.Header.EventID, Provider: corechannel.ProviderFeishu,
		ProviderEventID:   raw.EventV2Base.Header.EventID,
		Route:             corechannel.Route{AccountID: account.ID, ChatType: chatType, ChatID: stringValue(message.ChatId), ThreadID: stringValue(message.ThreadId)},
		Sender:            corechannel.SenderIdentity{ID: senderID, IsBot: stringValue(raw.Event.Sender.SenderType) != "user"},
		PlatformMessageID: messageID, Text: text, Reply: reply, Mentioned: mentioned, Media: media,
		OccurredAt: occurredAt, ReceivedAt: time.Now(),
	}, true
}

func mentionedBot(mentions []*larkim.MentionEvent, settings map[string]any) bool {
	if len(mentions) == 0 {
		return false
	}
	botOpenID, _ := settings["bot_open_id"].(string)
	if botOpenID == "" {
		return false
	}
	for _, mention := range mentions {
		if mention != nil && mention.Id != nil && stringValue(mention.Id.OpenId) == botOpenID {
			return true
		}
	}
	return false
}

// Send 使用 delivery_id 映射为飞书 uuid，避免确定性重试重复创建消息。
func (*Provider) Send(ctx context.Context, account corechannel.AccountConfig, message corechannel.OutboundMessage) (corechannel.Receipt, error) {
	appID, appSecret, err := credentials(account)
	if err != nil {
		return corechannel.Receipt{DeliveryID: message.DeliveryID, State: corechannel.DeliveryPermanent}, err
	}
	client := lark.NewClient(appID, appSecret)
	parts := corechannel.SplitText(message.Text, 30000)
	if len(parts) == 0 {
		return corechannel.Receipt{DeliveryID: message.DeliveryID, State: corechannel.DeliveryPermanent}, errors.New("feishu outbound text is empty")
	}
	lastMessageID := ""
	for index, part := range parts {
		body, _ := json.Marshal(map[string]string{"text": part})
		idempotencyKey := message.DeliveryID
		if index > 0 {
			idempotencyKey = fmt.Sprintf("%s-%d", message.DeliveryID, index)
		}
		if index == 0 && message.ReplyTo != nil && message.ReplyTo.MessageID != "" {
			req := larkim.NewReplyMessageReqBuilder().MessageId(message.ReplyTo.MessageID).Body(
				larkim.NewReplyMessageReqBodyBuilder().MsgType("text").Content(string(body)).Uuid(idempotencyKey).ReplyInThread(message.Route.ThreadID != "").Build(),
			).Build()
			resp, sendErr := client.Im.Message.Reply(ctx, req)
			if sendErr != nil {
				return failedReceipt(message.DeliveryID, lastMessageID, index, sendErr, 0, "")
			}
			if !resp.Success() {
				return failedReceipt(message.DeliveryID, lastMessageID, index, nil, resp.Code, resp.Msg)
			}
			lastMessageID = stringValue(resp.Data.MessageId)
			continue
		}
		req := larkim.NewCreateMessageReqBuilder().ReceiveIdType("chat_id").Body(
			larkim.NewCreateMessageReqBodyBuilder().ReceiveId(message.Route.ChatID).MsgType("text").Content(string(body)).Uuid(idempotencyKey).Build(),
		).Build()
		resp, sendErr := client.Im.Message.Create(ctx, req)
		if sendErr != nil {
			return failedReceipt(message.DeliveryID, lastMessageID, index, sendErr, 0, "")
		}
		if !resp.Success() {
			return failedReceipt(message.DeliveryID, lastMessageID, index, nil, resp.Code, resp.Msg)
		}
		lastMessageID = stringValue(resp.Data.MessageId)
	}
	return corechannel.Receipt{DeliveryID: message.DeliveryID, State: corechannel.DeliverySent, PlatformMessageID: lastMessageID}, nil
}

func failedReceipt(deliveryID, platformID string, index int, err error, code int, message string) (corechannel.Receipt, error) {
	state := corechannel.DeliveryTemporary
	if index > 0 {
		state = corechannel.DeliveryUnknown
	} else if code >= 230000 && code < 240000 {
		state = corechannel.DeliveryPermanent
	}
	if err == nil {
		err = fmt.Errorf("feishu API error %d: %s", code, message)
	}
	return corechannel.Receipt{DeliveryID: deliveryID, State: state, PlatformMessageID: platformID, ErrorCode: strconv.Itoa(code), ErrorMessage: err.Error()}, err
}

// SetTyping 对飞书首版静默降级。
func (*Provider) SetTyping(context.Context, corechannel.AccountConfig, corechannel.Route, bool) error {
	return nil
}

// DownloadMedia 通过飞书消息资源接口下载图片或文件。
func (*Provider) DownloadMedia(ctx context.Context, account corechannel.AccountConfig, media corechannel.MediaReference) (*corechannel.DownloadedMedia, error) {
	appID, appSecret, err := credentials(account)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimPrefix(media.URL, "feishu://"), "/")
	if len(parts) != 2 || parts[0] == "" {
		return nil, errors.New("invalid feishu media reference")
	}
	resourceType := parts[1]
	client := lark.NewClient(appID, appSecret)
	req := larkim.NewGetMessageResourceReqBuilder().MessageId(parts[0]).FileKey(media.ID).Type(resourceType).Build()
	resp, err := client.Im.MessageResource.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success() {
		return nil, fmt.Errorf("feishu media API error %d: %s", resp.Code, resp.Msg)
	}
	name := media.FileName
	if name == "" {
		name = resp.FileName
	}
	return &corechannel.DownloadedMedia{Body: io.NopCloser(resp.File), MIMEType: media.MIMEType, FileName: name, Size: media.Size}, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
