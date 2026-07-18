package gateway

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	corellm "github.com/boxify/api-go/internal/core/llm"
	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

const (
	pairingTTL       = time.Hour
	maxPendingPairs  = 3
	maxExternalRunes = 64 * 1024
)

var errGatewayEnqueue = errors.New("gateway task enqueue failed")

// HandleInbound 按持久化去重、权限、路由和调度顺序处理规范化事件。
func (s *Service) HandleInbound(ctx context.Context, account *models.ChannelAccount, event corechannel.InboundEvent) (*models.ChannelInboxEvent, bool, error) {
	if s == nil || s.svc == nil || s.svc.ChannelGatewayRepo == nil || account == nil {
		return nil, false, xerr.Internal("网关入站服务未初始化", nil)
	}
	event.Route.AccountID = account.ID.String()
	event.Provider = corechannel.ProviderName(account.Provider)
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now()
	}
	if err := event.Validate(); err != nil {
		return nil, false, xerr.BadRequest("渠道事件格式无效")
	}
	normalized, err := eventJSON(event)
	if err != nil {
		return nil, false, err
	}
	row, created, err := s.svc.ChannelGatewayRepo.CreateInboxEvent(ctx, &models.ChannelInboxEvent{
		UserID: account.UserID, AccountID: account.ID, ProviderEventID: event.ProviderEventID,
		RouteKey: event.Route.Key(), NormalizedEvent: normalized, Status: models.ChannelInboxStatusReceived,
	})
	if err != nil || !created {
		if row != nil && s.log != nil {
			s.log.InfoContext(ctx, "网关入站事件去重完成", "inbox_id", row.ID, "account_id", account.ID, "provider", account.Provider, "duplicate", !created)
		}
		return row, created, err
	}
	if s.log != nil {
		s.log.InfoContext(ctx, "网关入站事件已持久化", "inbox_id", row.ID, "account_id", account.ID, "provider", account.Provider)
	}
	if err := s.processPersistedInbound(ctx, account, row, event); err != nil {
		status := models.ChannelInboxStatusFailed
		if errors.Is(err, errGatewayEnqueue) {
			// 保持 received，周期对账会从已落库消息继续投递，不重复创建用户消息。
			status = models.ChannelInboxStatusReceived
		}
		_ = s.svc.ChannelGatewayRepo.UpdateInboxEvent(ctx, row.ID, map[string]any{"status": status, "error_message": safeGatewayError(err)})
		return row, true, err
	}
	updated, findErr := s.svc.ChannelGatewayRepo.FindInboxEventByID(ctx, row.ID)
	if findErr == nil {
		row = updated
	}
	if s.log != nil {
		s.log.InfoContext(ctx, "网关入站事件处理完成", "inbox_id", row.ID, "account_id", account.ID, "status", row.Status)
	}
	return row, true, nil
}

// RecoverInbox 重新投递已落库但尚未成功入队的事件。
func (s *Service) RecoverInbox(ctx context.Context, limit int) error {
	rows, err := s.svc.ChannelGatewayRepo.ListRecoverableInboxEvents(ctx, limit)
	if err != nil {
		return err
	}
	for _, row := range rows {
		account, findErr := s.svc.ChannelGatewayRepo.FindAccountByID(ctx, row.UserID, row.AccountID)
		if findErr != nil || !account.Enabled {
			continue
		}
		var event corechannel.InboundEvent
		data, _ := json.Marshal(row.NormalizedEvent)
		if unmarshalErr := json.Unmarshal(data, &event); unmarshalErr != nil {
			_ = s.svc.ChannelGatewayRepo.UpdateInboxEvent(ctx, row.ID, map[string]any{"status": models.ChannelInboxStatusFailed, "error_message": "渠道事件无法恢复"})
			continue
		}
		if processErr := s.processPersistedInbound(ctx, account, row, event); processErr != nil {
			continue
		}
	}
	return nil
}

func (s *Service) processPersistedInbound(ctx context.Context, account *models.ChannelAccount, inbox *models.ChannelInboxEvent, event corechannel.InboundEvent) error {
	if event.Sender.IsBot {
		return s.markInbox(ctx, inbox.ID, models.ChannelInboxStatusIgnored, "")
	}
	if inbox.CoveMessageID != nil && inbox.ConversationID != nil {
		return s.enqueueTurn(ctx, inbox.ID)
	}
	binding, allowed, err := s.authorizeAndBind(ctx, account, inbox, event)
	if err != nil || !allowed {
		return err
	}
	if binding.RequireMention && event.Route.ChatType == corechannel.ChatTypeGroup && !event.Mentioned {
		return s.markInbox(ctx, inbox.ID, models.ChannelInboxStatusIgnored, "")
	}
	messageID := inboundUserMessageID(inbox.ID)
	if existing, findErr := s.svc.MessageRepo.FindByID(ctx, account.UserID, messageID); findErr == nil {
		if strings.TrimSpace(event.Text) == "" && len(event.Media) > 0 && !hasExtractedAttachment(existing.MetaData) {
			_ = s.queueReply(ctx, account, inbox, event.Route, "我已保存附件，但暂时无法理解其中内容。请补充一段文字说明后再试。")
			now := time.Now()
			return s.svc.ChannelGatewayRepo.UpdateInboxEvent(ctx, inbox.ID, map[string]any{
				"conversation_id": existing.ConversationID, "cove_message_id": existing.ID,
				"status": models.ChannelInboxStatusFailed, "error_message": "纯媒体内容无法解析", "processed_at": &now,
			})
		}
		agentConfigID, resolveErr := s.resolveAgentConfig(ctx, account, binding)
		if resolveErr != nil {
			return resolveErr
		}
		if err := s.svc.ChannelGatewayRepo.UpdateInboxEvent(ctx, inbox.ID, map[string]any{
			"conversation_id": existing.ConversationID, "cove_message_id": existing.ID,
			"resolved_agent_config_id": agentConfigID, "tool_policy": binding.ToolPolicy,
		}); err != nil {
			return err
		}
		return s.enqueueTurn(ctx, inbox.ID)
	} else if xerr.From(findErr).Kind != xerr.KindNotFound {
		return findErr
	}
	attachmentMeta, attachments, err := s.ProcessMedia(ctx, account, event.Media)
	if err != nil {
		_ = s.queueReply(ctx, account, inbox, event.Route, "附件下载或安全校验失败，请检查文件类型、大小和来源。")
		return err
	}
	if strings.TrimSpace(event.Text) == "" && len(event.Media) > 0 && len(attachments) == 0 {
		conversationID, conversationErr := s.ensureBindingConversation(ctx, account, binding, event)
		if conversationErr != nil {
			return conversationErr
		}
		message, saveErr := s.svc.MessageRepo.Create(ctx, account.UserID, &models.Message{
			ID: messageID, ConversationID: conversationID, Role: string(corellm.UserRole),
			Content:  "<external_untrusted>\n[附件内容无法解析]\n</external_untrusted>",
			MetaData: &models.MessageMetaData{SenderName: event.Sender.DisplayName, Attachments: attachmentMeta},
		})
		if saveErr != nil {
			message, saveErr = s.svc.MessageRepo.FindByID(ctx, account.UserID, messageID)
			if saveErr != nil {
				return saveErr
			}
		}
		_ = s.queueReply(ctx, account, inbox, event.Route, "我已保存附件，但暂时无法理解其中内容。请补充一段文字说明后再试。")
		now := time.Now()
		return s.svc.ChannelGatewayRepo.UpdateInboxEvent(ctx, inbox.ID, map[string]any{
			"conversation_id": message.ConversationID, "cove_message_id": message.ID,
			"status": models.ChannelInboxStatusFailed, "error_message": "纯媒体内容无法解析", "processed_at": &now,
		})
	}
	conversationID, err := s.ensureBindingConversation(ctx, account, binding, event)
	if err != nil {
		return err
	}
	agentConfigID, err := s.resolveAgentConfig(ctx, account, binding)
	if err != nil {
		return err
	}
	imageKeys := make([]string, 0)
	for _, attachment := range attachmentMeta {
		if attachment.Kind == "image" {
			imageKeys = append(imageKeys, attachment.StorageKey)
		}
	}
	message, err := s.svc.MessageRepo.Create(ctx, account.UserID, &models.Message{
		ID: messageID, ConversationID: conversationID, Role: string(corellm.UserRole), Content: untrustedMessage(event),
		MetaData: &models.MessageMetaData{ImageKeys: imageKeys, SenderName: event.Sender.DisplayName, Attachments: attachmentMeta},
	})
	if err != nil {
		// 并发恢复可能已经写入同一确定性消息，读取现有记录继续推进 Inbox。
		message, err = s.svc.MessageRepo.FindByID(ctx, account.UserID, messageID)
		if err != nil {
			return err
		}
	}
	values := map[string]any{
		"conversation_id": conversationID, "cove_message_id": message.ID,
		"resolved_agent_config_id": agentConfigID, "tool_policy": binding.ToolPolicy, "status": models.ChannelInboxStatusReceived,
	}
	if err := s.svc.ChannelGatewayRepo.UpdateInboxEvent(ctx, inbox.ID, values); err != nil {
		return err
	}
	_ = attachments // 提取文本已保存在 Message metadata，Worker 会重建临时附件。
	return s.enqueueTurn(ctx, inbox.ID)
}

func (s *Service) authorizeAndBind(ctx context.Context, account *models.ChannelAccount, inbox *models.ChannelInboxEvent, event corechannel.InboundEvent) (*models.ChannelBinding, bool, error) {
	if event.Route.ChatType == corechannel.ChatTypeDirect {
		allowed, err := s.authorizeDirect(ctx, account, inbox, event)
		if err != nil || !allowed {
			return nil, false, err
		}
	}
	binding, err := s.svc.ChannelGatewayRepo.FindBindingByRoute(ctx, account.ID, event.Route.Key())
	if err == nil {
		if !binding.Enabled {
			return binding, false, s.markInbox(ctx, inbox.ID, models.ChannelInboxStatusIgnored, "")
		}
		return binding, true, nil
	}
	if xerr.From(err).Kind != xerr.KindNotFound {
		return nil, false, err
	}
	if event.Route.ChatType == corechannel.ChatTypeGroup {
		_, createErr := s.svc.ChannelGatewayRepo.CreateBinding(ctx, account.UserID, &models.ChannelBinding{
			AccountID: account.ID, RouteKey: event.Route.Key(), ChatType: string(event.Route.ChatType),
			ExternalChatID: event.Route.ChatID, ExternalThreadID: event.Route.ThreadID,
			RequireMention: true, ToolPolicy: models.ChannelToolPolicySafe, Enabled: false,
		})
		if createErr != nil {
			if existing, findErr := s.svc.ChannelGatewayRepo.FindBindingByRoute(ctx, account.ID, event.Route.Key()); findErr == nil && existing.Enabled {
				return existing, true, nil
			}
		}
		return nil, false, s.markInbox(ctx, inbox.ID, models.ChannelInboxStatusIgnored, "")
	}
	binding, err = s.svc.ChannelGatewayRepo.CreateBinding(ctx, account.UserID, &models.ChannelBinding{
		AccountID: account.ID, RouteKey: event.Route.Key(), ChatType: string(event.Route.ChatType),
		ExternalChatID: event.Route.ChatID, ExternalThreadID: event.Route.ThreadID,
		RequireMention: false, ToolPolicy: models.ChannelToolPolicyInherit, Enabled: true,
	})
	if err != nil {
		existing, findErr := s.svc.ChannelGatewayRepo.FindBindingByRoute(ctx, account.ID, event.Route.Key())
		return existing, findErr == nil && existing.Enabled, findErr
	}
	return binding, true, nil
}

func (s *Service) authorizeDirect(ctx context.Context, account *models.ChannelAccount, inbox *models.ChannelInboxEvent, event corechannel.InboundEvent) (bool, error) {
	identity, err := s.svc.ChannelGatewayRepo.FindIdentity(ctx, account.ID, event.Sender.ID, event.Route.ChatID)
	if err == nil {
		switch identity.Status {
		case models.ChannelIdentityStatusAllowed:
			return true, nil
		case models.ChannelIdentityStatusBlocked:
			return false, s.markInbox(ctx, inbox.ID, models.ChannelInboxStatusIgnored, "")
		case models.ChannelIdentityStatusPending:
			if identity.PairingExpiresAt != nil && identity.PairingExpiresAt.After(time.Now()) {
				_ = s.queueReply(ctx, account, inbox, event.Route, "配对请求仍在等待批准（验证码尾号 "+identity.PairingCodeMasked+"）。")
				return false, s.markInbox(ctx, inbox.ID, models.ChannelInboxStatusPendingPair, "")
			}
			return s.refreshPairing(ctx, account, inbox, event, identity)
		}
	}
	if err != nil && xerr.From(err).Kind != xerr.KindNotFound {
		return false, err
	}
	count, err := s.svc.ChannelGatewayRepo.CountPendingIdentities(ctx, account.ID)
	if err != nil {
		return false, err
	}
	if count >= maxPendingPairs {
		_ = s.queueReply(ctx, account, inbox, event.Route, "该账号待处理的配对请求已达上限，请联系账号所有者。")
		return false, s.markInbox(ctx, inbox.ID, models.ChannelInboxStatusPendingPair, "")
	}
	code, hash, expiresAt, err := pairingCode(account.ID)
	if err != nil {
		return false, err
	}
	identity, err = s.svc.ChannelGatewayRepo.CreateIdentity(ctx, &models.ChannelIdentity{
		AccountID: account.ID, ExternalUserID: event.Sender.ID, ExternalChatID: event.Route.ChatID,
		DisplayName: event.Sender.DisplayName, Status: models.ChannelIdentityStatusPending,
		PairingCodeHash: hash, PairingCodeMasked: code[len(code)-2:], PairingExpiresAt: &expiresAt,
	})
	if err != nil {
		return false, err
	}
	_ = identity
	_ = s.queueReply(ctx, account, inbox, event.Route, fmt.Sprintf("Cove 配对码：%s。请在 1 小时内让账号所有者批准此配对请求。", code))
	return false, s.markInbox(ctx, inbox.ID, models.ChannelInboxStatusPendingPair, "")
}

func (s *Service) refreshPairing(ctx context.Context, account *models.ChannelAccount, inbox *models.ChannelInboxEvent, event corechannel.InboundEvent, identity *models.ChannelIdentity) (bool, error) {
	code, hash, expiresAt, err := pairingCode(account.ID)
	if err != nil {
		return false, err
	}
	_, err = s.svc.ChannelGatewayRepo.UpdateIdentity(ctx, account.UserID, identity.ID, map[string]any{
		"pairing_code_hash": hash, "pairing_code_masked": code[len(code)-2:], "pairing_expires_at": &expiresAt,
	})
	if err != nil {
		return false, err
	}
	_ = s.queueReply(ctx, account, inbox, event.Route, fmt.Sprintf("新的 Cove 配对码：%s。请在 1 小时内完成批准。", code))
	return false, s.markInbox(ctx, inbox.ID, models.ChannelInboxStatusPendingPair, "")
}

func (s *Service) ensureBindingConversation(ctx context.Context, account *models.ChannelAccount, binding *models.ChannelBinding, event corechannel.InboundEvent) (uuid.UUID, error) {
	if binding.ConversationID != nil {
		if _, err := s.svc.ConversationRepo.FindByID(ctx, account.UserID, *binding.ConversationID); err == nil {
			return *binding.ConversationID, nil
		}
	}
	title := strings.TrimSpace(event.Sender.DisplayName)
	if event.Route.ChatType == corechannel.ChatTypeGroup || title == "" {
		title = string(event.Provider) + " · " + redactExternalID(event.Route.ChatID)
	}
	conversation, err := s.svc.ConversationRepo.Create(ctx, account.UserID, &models.Conversation{Title: truncateRunes(title, 255)})
	if err != nil {
		return uuid.Nil, err
	}
	updated, err := s.svc.ChannelGatewayRepo.UpdateBinding(ctx, account.UserID, binding.ID, map[string]any{"conversation_id": conversation.ID})
	if err != nil {
		return uuid.Nil, err
	}
	binding.ConversationID = updated.ConversationID
	return conversation.ID, nil
}

func (s *Service) resolveAgentConfig(ctx context.Context, account *models.ChannelAccount, binding *models.ChannelBinding) (*uuid.UUID, error) {
	if selected := selectedAgentConfigID(account, binding); selected != nil {
		if _, err := s.svc.AgentConfigRepo.FindByID(ctx, account.UserID, *selected); err != nil {
			return nil, err
		}
		return selected, nil
	}
	config, err := s.svc.AgentConfigRepo.FindDefault(ctx, account.UserID)
	if err != nil {
		if xerr.From(err).Kind == xerr.KindNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config.ID, nil
}

func selectedAgentConfigID(account *models.ChannelAccount, binding *models.ChannelBinding) *uuid.UUID {
	if binding != nil && binding.AgentConfigID != nil {
		return binding.AgentConfigID
	}
	if account != nil {
		return account.DefaultAgentConfigID
	}
	return nil
}

func (s *Service) enqueueTurn(ctx context.Context, inboxID uuid.UUID) error {
	task, err := types.NewGatewayTurnTask(inboxID)
	if err != nil {
		return err
	}
	if s.svc.TaskProducer == nil {
		return fmt.Errorf("%w: producer is unavailable", errGatewayEnqueue)
	}
	info, err := s.svc.TaskProducer.Enqueue(ctx, task)
	if err != nil {
		return fmt.Errorf("%w: %v", errGatewayEnqueue, err)
	}
	now := time.Now()
	if s.log != nil {
		s.log.InfoContext(ctx, "网关 Agent 回合已入队", "inbox_id", inboxID, "task_id", info.ID)
	}
	return s.svc.ChannelGatewayRepo.UpdateInboxEvent(ctx, inboxID, map[string]any{
		"status": models.ChannelInboxStatusQueued, "task_id": info.ID, "enqueued_at": &now, "error_message": "",
	})
}

func (s *Service) queueReply(ctx context.Context, account *models.ChannelAccount, inbox *models.ChannelInboxEvent, route corechannel.Route, text string) error {
	deliveryID := uuid.NewString()
	outbox, err := s.svc.ChannelGatewayRepo.CreateOutboxMessage(ctx, &models.ChannelOutboxMessage{
		UserID: account.UserID, AccountID: account.ID, InboxEventID: inbox.ID,
		DeliveryID: deliveryID, Route: routeJSON(route), ReplyTo: inboxReplyJSON(inbox.NormalizedEvent),
		Content: text, Status: models.ChannelOutboxStatusPending,
	})
	if err != nil {
		return err
	}
	task, err := types.NewGatewayDeliverTask(outbox.ID)
	if err != nil {
		return err
	}
	if s.svc.TaskProducer != nil {
		_, err = s.svc.TaskProducer.Enqueue(ctx, task)
	}
	if err != nil {
		if s.log != nil {
			s.log.WarnContext(ctx, "网关 Outbox 即时入队失败，将由对账恢复", "outbox_id", outbox.ID, "account_id", account.ID)
		}
		return nil
	}
	if s.log != nil {
		s.log.InfoContext(ctx, "网关最终回复已写入 Outbox", "inbox_id", inbox.ID, "outbox_id", outbox.ID, "account_id", account.ID)
	}
	return nil
}

func (s *Service) markInbox(ctx context.Context, id uuid.UUID, status, errorMessage string) error {
	values := map[string]any{"status": status, "error_message": errorMessage}
	if status == models.ChannelInboxStatusIgnored || status == models.ChannelInboxStatusFailed || status == models.ChannelInboxStatusCompleted {
		now := time.Now()
		values["processed_at"] = &now
	}
	return s.svc.ChannelGatewayRepo.UpdateInboxEvent(ctx, id, values)
}

func pairingCode(accountID uuid.UUID) (string, string, time.Time, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", "", time.Time{}, err
	}
	code := fmt.Sprintf("%06d", value.Int64())
	sum := sha256.Sum256([]byte(accountID.String() + ":" + code))
	return code, hex.EncodeToString(sum[:]), time.Now().Add(pairingTTL), nil
}

func inboundUserMessageID(inboxID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("cove:gateway:user:"+inboxID.String()))
}

func hasExtractedAttachment(meta *models.MessageMetaData) bool {
	if meta == nil {
		return false
	}
	for _, attachment := range meta.Attachments {
		if strings.TrimSpace(attachment.ExtractedText) != "" {
			return true
		}
	}
	return false
}

func eventJSON(event corechannel.InboundEvent) (models.JSONMap, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	out := models.JSONMap{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func routeJSON(route corechannel.Route) models.JSONMap {
	return models.JSONMap{"account_id": route.AccountID, "chat_type": string(route.ChatType), "chat_id": route.ChatID, "thread_id": route.ThreadID}
}

func inboxReplyJSON(event models.JSONMap) models.JSONMap {
	messageID, _ := event["platform_message_id"].(string)
	if strings.TrimSpace(messageID) == "" {
		return nil
	}
	return models.JSONMap{"message_id": messageID}
}

func untrustedMessage(event corechannel.InboundEvent) string {
	text := truncateRunes(strings.TrimSpace(event.Text), maxExternalRunes)
	parts := []string{"<external_untrusted>"}
	if event.Reply != nil && strings.TrimSpace(event.Reply.Text) != "" {
		parts = append(parts, "Reply reference (untrusted): "+truncateRunes(event.Reply.Text, 2000))
	}
	if text != "" {
		parts = append(parts, text)
	}
	parts = append(parts, "</external_untrusted>")
	return strings.Join(parts, "\n")
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func redactExternalID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}

func safeGatewayError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "处理已取消"
	}
	return "渠道事件处理失败"
}
