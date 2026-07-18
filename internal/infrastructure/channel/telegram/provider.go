// Package telegram 实现 Telegram Bot 长轮询适配器。
package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// Provider 使用 go-telegram/bot 官方 API 封装。
type Provider struct{ httpClient *http.Client }

// New 创建 Telegram Provider。
func New(client *http.Client) *Provider {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Provider{httpClient: client}
}

// Descriptor 返回 Telegram 凭据表单和能力矩阵。
func (*Provider) Descriptor() corechannel.ProviderDescriptor {
	return corechannel.ProviderDescriptor{
		Name: corechannel.ProviderTelegram, DisplayName: "Telegram",
		Description: "通过 Bot API 长轮询接入 Telegram 私聊、群聊和话题",
		CredentialFields: []corechannel.FieldDescriptor{
			{Key: "bot_token", Label: "Bot Token", Type: "password", Required: true, Sensitive: true},
		},
		SettingFields: []corechannel.FieldDescriptor{
			{Key: "bot_username", Label: "Bot 用户名", Type: "text", Description: "用于群聊提及识别"},
		},
		Capabilities: corechannel.Capabilities{
			DirectMessages: true, GroupMessages: true, Threads: true, Replies: true,
			Mentions: true, Typing: true, InboundImages: true, InboundFiles: true, OutboundText: true,
		},
		MaxTextLength: 4096,
	}
}

func (p *Provider) newBot(account corechannel.AccountConfig, options ...tgbot.Option) (*tgbot.Bot, error) {
	token := strings.TrimSpace(account.Credentials["bot_token"])
	if token == "" {
		return nil, errors.New("telegram bot token is required")
	}
	options = append(options, tgbot.WithHTTPClient(30*time.Second, p.httpClient))
	return tgbot.New(token, options...)
}

// TestAccount 通过 getMe 验证 Bot Token。
func (p *Provider) TestAccount(ctx context.Context, account corechannel.AccountConfig) error {
	bot, err := p.newBot(account)
	if err != nil {
		return err
	}
	_, err = bot.GetMe(ctx)
	return err
}

// Receive 启动账号独占的 Telegram 长轮询，直到租约上下文取消。
func (p *Provider) Receive(ctx context.Context, account corechannel.AccountConfig, handler corechannel.EventHandler) error {
	bot, err := p.newBot(account, tgbot.WithDefaultHandler(func(eventCtx context.Context, bot *tgbot.Bot, update *tgmodels.Update) {
		event, ok := normalize(account, bot, update)
		if !ok || handler == nil {
			return
		}
		_ = handler.HandleInbound(eventCtx, event)
	}))
	if err != nil {
		return err
	}
	bot.Start(ctx)
	return nil
}

func normalize(account corechannel.AccountConfig, bot *tgbot.Bot, update *tgmodels.Update) (corechannel.InboundEvent, bool) {
	if update == nil || update.Message == nil || update.Message.From == nil {
		return corechannel.InboundEvent{}, false
	}
	message := update.Message
	chatType := corechannel.ChatTypeGroup
	if message.Chat.Type == tgmodels.ChatTypePrivate {
		chatType = corechannel.ChatTypeDirect
	}
	text := message.Text
	if text == "" {
		text = message.Caption
	}
	threadID := ""
	if message.MessageThreadID != 0 {
		threadID = strconv.Itoa(message.MessageThreadID)
	}
	media := make([]corechannel.MediaReference, 0, 2)
	if len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		media = append(media, corechannel.MediaReference{ID: photo.FileID, Kind: "image", MIMEType: "image/jpeg", Size: int64(photo.FileSize)})
	}
	if message.Document != nil {
		media = append(media, corechannel.MediaReference{
			ID: message.Document.FileID, Kind: "document", MIMEType: message.Document.MimeType,
			FileName: message.Document.FileName, Size: message.Document.FileSize,
		})
	}
	var reply *corechannel.ReplyReference
	if message.ReplyToMessage != nil {
		replyText := message.ReplyToMessage.Text
		if replyText == "" {
			replyText = message.ReplyToMessage.Caption
		}
		reply = &corechannel.ReplyReference{MessageID: strconv.Itoa(message.ReplyToMessage.ID), Text: replyText}
	}
	username, _ := account.Settings["bot_username"].(string)
	mentioned := chatType == corechannel.ChatTypeDirect || containsMention(text, username)
	return corechannel.InboundEvent{
		ID: strconv.FormatInt(update.ID, 10), Provider: corechannel.ProviderTelegram,
		ProviderEventID: strconv.FormatInt(update.ID, 10),
		Route:           corechannel.Route{AccountID: account.ID, ChatType: chatType, ChatID: strconv.FormatInt(message.Chat.ID, 10), ThreadID: threadID},
		Sender: corechannel.SenderIdentity{
			ID: strconv.FormatInt(message.From.ID, 10), DisplayName: strings.TrimSpace(message.From.FirstName + " " + message.From.LastName),
			Username: message.From.Username, IsBot: message.From.IsBot,
		},
		PlatformMessageID: strconv.Itoa(message.ID), Text: text, Reply: reply, Mentioned: mentioned, Media: media,
		OccurredAt: time.Unix(int64(message.Date), 0), ReceivedAt: time.Now(),
	}, true
}

func containsMention(text, username string) bool {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	return username != "" && strings.Contains(strings.ToLower(text), "@"+strings.ToLower(username))
}

// Send 按 4096 字符限长顺序发送最终文本。
func (p *Provider) Send(ctx context.Context, account corechannel.AccountConfig, message corechannel.OutboundMessage) (corechannel.Receipt, error) {
	bot, err := p.newBot(account)
	if err != nil {
		return corechannel.Receipt{DeliveryID: message.DeliveryID, State: corechannel.DeliveryPermanent}, err
	}
	chatID := any(message.Route.ChatID)
	if parsed, parseErr := strconv.ParseInt(message.Route.ChatID, 10, 64); parseErr == nil {
		chatID = parsed
	}
	threadID, _ := strconv.Atoi(message.Route.ThreadID)
	parts := corechannel.SplitText(message.Text, 4096)
	if len(parts) == 0 {
		return corechannel.Receipt{DeliveryID: message.DeliveryID, State: corechannel.DeliveryPermanent}, errors.New("telegram outbound text is empty")
	}
	lastMessageID := ""
	for index, part := range parts {
		params := &tgbot.SendMessageParams{ChatID: chatID, MessageThreadID: threadID, Text: part}
		if index == 0 && message.ReplyTo != nil {
			if replyID, parseErr := strconv.Atoi(message.ReplyTo.MessageID); parseErr == nil {
				params.ReplyParameters = &tgmodels.ReplyParameters{MessageID: replyID, AllowSendingWithoutReply: true}
			}
		}
		sent, sendErr := bot.SendMessage(ctx, params)
		if sendErr != nil {
			state := telegramDeliveryState(sendErr, index)
			return corechannel.Receipt{DeliveryID: message.DeliveryID, State: state, PlatformMessageID: lastMessageID, ErrorMessage: sendErr.Error()}, sendErr
		}
		lastMessageID = strconv.Itoa(sent.ID)
	}
	return corechannel.Receipt{DeliveryID: message.DeliveryID, State: corechannel.DeliverySent, PlatformMessageID: lastMessageID}, nil
}

func telegramDeliveryState(err error, partIndex int) corechannel.DeliveryState {
	if partIndex > 0 {
		return corechannel.DeliveryUnknown
	}
	if tgbot.IsTooManyRequestsError(err) {
		return corechannel.DeliveryTemporary
	}
	if errors.Is(err, tgbot.ErrorForbidden) || errors.Is(err, tgbot.ErrorBadRequest) ||
		errors.Is(err, tgbot.ErrorUnauthorized) || errors.Is(err, tgbot.ErrorNotFound) {
		return corechannel.DeliveryPermanent
	}
	// 网络错误可能发生在请求写出之后，不能盲目重发第一段。
	return corechannel.DeliveryUnknown
}

// SetTyping 发送 Telegram typing 动作；停止动作由平台自动过期。
func (p *Provider) SetTyping(ctx context.Context, account corechannel.AccountConfig, route corechannel.Route, active bool) error {
	if !active {
		return nil
	}
	bot, err := p.newBot(account)
	if err != nil {
		return err
	}
	chatID := any(route.ChatID)
	if parsed, parseErr := strconv.ParseInt(route.ChatID, 10, 64); parseErr == nil {
		chatID = parsed
	}
	threadID, _ := strconv.Atoi(route.ThreadID)
	_, err = bot.SendChatAction(ctx, &tgbot.SendChatActionParams{ChatID: chatID, MessageThreadID: threadID, Action: tgmodels.ChatActionTyping})
	return err
}

// DownloadMedia 通过 Telegram GetFile API 获取媒体，不把含 Token 的 URL 持久化。
func (p *Provider) DownloadMedia(ctx context.Context, account corechannel.AccountConfig, media corechannel.MediaReference) (*corechannel.DownloadedMedia, error) {
	bot, err := p.newBot(account)
	if err != nil {
		return nil, err
	}
	file, err := bot.GetFile(ctx, &tgbot.GetFileParams{FileID: media.ID})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bot.FileDownloadLink(file), nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("telegram media download returned %s", resp.Status)
	}
	size := resp.ContentLength
	if size <= 0 {
		size = media.Size
	}
	mimeType := media.MIMEType
	if mimeType == "" {
		mimeType = resp.Header.Get("Content-Type")
	}
	return &corechannel.DownloadedMedia{Body: io.NopCloser(resp.Body), MIMEType: mimeType, FileName: media.FileName, Size: size}, nil
}
