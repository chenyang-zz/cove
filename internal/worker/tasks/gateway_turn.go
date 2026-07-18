package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	flowchat "github.com/boxify/api-go/internal/domain/flow/chat"
	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/infrastructure/lease"
	chatlogic "github.com/boxify/api-go/internal/logic/chat"
	gatewaylogic "github.com/boxify/api-go/internal/logic/gateway"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// GatewayTurnTask 串行执行一个 Conversation 的外部聊天回合。
type GatewayTurnTask struct {
	svc *svc.ServiceContext
	log *slog.Logger
}

// NewGatewayTurnTask 创建网关回合 Worker。
func NewGatewayTurnTask(svcCtx *svc.ServiceContext) *GatewayTurnTask {
	return &GatewayTurnTask{svc: svcCtx, log: xlog.Component("worker.gateway.turn")}
}

// Handle 执行最终生成并把 Assistant 文本写入 Outbox。
func (h *GatewayTurnTask) Handle(ctx context.Context, task *types.Task) error {
	startedAt := time.Now()
	logger := h.logger()

	// 任务只携带稳定 Inbox ID；后续状态和路由均以数据库快照为准，避免消费旧 payload。
	payload, err := gatewayTurnPayload(task)
	if err != nil {
		logger.WarnContext(ctx, "网关 Agent 回合任务载荷无效",
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return skipRetry(err)
	}
	logger.InfoContext(ctx, "开始执行网关 Agent 回合",
		slog.String("inbox_id", payload.InboxEventID.String()),
	)

	inbox, err := h.svc.ChannelGatewayRepo.FindInboxEventByID(ctx, payload.InboxEventID)
	if err != nil {
		logger.WarnContext(ctx, "读取网关 Inbox 失败，停止回合",
			slog.String("inbox_id", payload.InboxEventID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return skipRetry(err)
	}

	// 终态 Inbox 必须幂等退出；completed 仅恢复 Outbox 入队，不再次调用 Agent。
	switch inbox.Status {
	case models.ChannelInboxStatusIgnored, models.ChannelInboxStatusPendingPair, models.ChannelInboxStatusFailed:
		logger.InfoContext(ctx, "网关 Agent 回合命中无需处理状态",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("account_id", inbox.AccountID.String()),
			slog.String("inbox_status", inbox.Status),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return nil
	case models.ChannelInboxStatusCompleted:
		logger.InfoContext(ctx, "网关 Agent 回合已完成，恢复 Outbox 投递",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("account_id", inbox.AccountID.String()),
		)
		return h.enqueueExistingOutbox(ctx, inbox.ID)
	}
	if inbox.ConversationID == nil || inbox.CoveMessageID == nil {
		logger.WarnContext(ctx, "网关 Inbox 缺少回合上下文",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("account_id", inbox.AccountID.String()),
			slog.Bool("has_conversation", inbox.ConversationID != nil),
			slog.Bool("has_user_message", inbox.CoveMessageID != nil),
		)
		return skipRetry(errors.New("gateway inbox is missing conversation or user message"))
	}

	// 同一 Conversation 只允许一个可续租回合，避免多个渠道事件并发改写上下文和消息顺序。
	lock, acquired, err := lease.Acquire(ctx, h.svc.Redis.Raw(), "gateway:turn:lock:"+inbox.ConversationID.String(), 2*time.Minute)
	if err != nil {
		logger.WarnContext(ctx, "获取网关会话锁失败",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("conversation_id", inbox.ConversationID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return err
	}
	if !acquired {
		logger.InfoContext(ctx, "网关会话正在处理其他回合，等待任务重试",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("conversation_id", inbox.ConversationID.String()),
		)
		return errors.New("gateway conversation is busy")
	}
	logger.InfoContext(ctx, "网关会话锁已获取",
		slog.String("inbox_id", inbox.ID.String()),
		slog.String("conversation_id", inbox.ConversationID.String()),
	)

	runCtx, stopKeepAlive := lock.KeepAlive(ctx)
	defer func() {
		stopKeepAlive()
		if releaseErr := lock.Release(context.Background()); releaseErr != nil {
			logger.Warn("释放网关会话锁失败，等待租约自动过期",
				slog.String("inbox_id", inbox.ID.String()),
				slog.String("conversation_id", inbox.ConversationID.String()),
				slog.String("error_kind", gatewayTurnErrorKind(releaseErr)),
			)
		}
	}()

	// Assistant ID 由 Inbox 确定性生成；Worker 重启后可识别已落库结果并只补齐 Outbox。
	assistantID := uuid.NewSHA1(uuid.NameSpaceOID, inbox.ID[:])
	if existing, findErr := h.svc.MessageRepo.FindByID(runCtx, inbox.UserID, assistantID); findErr == nil {
		logger.InfoContext(runCtx, "检测到已落库 Assistant，跳过重复生成",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("conversation_id", inbox.ConversationID.String()),
			slog.String("assistant_message_id", existing.ID.String()),
		)
		if finishErr := h.finishTurn(runCtx, inbox, existing); finishErr != nil {
			return finishErr
		}
		logger.InfoContext(runCtx, "网关 Agent 回合恢复完成",
			slog.String("inbox_id", inbox.ID.String()),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return nil
	} else if xerr.From(findErr).Kind != xerr.KindNotFound {
		logger.WarnContext(runCtx, "查询确定性 Assistant 消息失败",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("assistant_message_id", assistantID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(findErr)),
		)
		return findErr
	}

	// 在调用模型前先持久化 processing，便于观察正在运行的回合和故障恢复。
	if err := h.svc.ChannelGatewayRepo.UpdateInboxEvent(runCtx, inbox.ID, map[string]any{"status": models.ChannelInboxStatusProcessing}); err != nil {
		logger.WarnContext(runCtx, "更新网关 Inbox 处理状态失败",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return err
	}
	logger.InfoContext(runCtx, "网关 Inbox 已进入 Agent 处理状态",
		slog.String("inbox_id", inbox.ID.String()),
		slog.String("conversation_id", inbox.ConversationID.String()),
	)

	// 从租户隔离的 Account 重新加载 Provider 配置，不信任入站事件内携带的账号信息。
	account, err := h.svc.ChannelGatewayRepo.FindAccountByID(runCtx, inbox.UserID, inbox.AccountID)
	if err != nil {
		logger.WarnContext(runCtx, "读取网关渠道账号失败",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("account_id", inbox.AccountID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return err
	}
	var event corechannel.InboundEvent
	data, _ := json.Marshal(inbox.NormalizedEvent)
	if err := json.Unmarshal(data, &event); err != nil {
		logger.WarnContext(runCtx, "解析网关规范化事件失败",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("account_id", inbox.AccountID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return skipRetry(err)
	}
	provider, ok := h.svc.ChannelRegistry.Get(corechannel.ProviderName(account.Provider))
	if !ok {
		logger.WarnContext(runCtx, "网关渠道 Provider 不可用",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("account_id", inbox.AccountID.String()),
			slog.String("provider", account.Provider),
		)
		return skipRetry(errors.New("gateway provider is unavailable"))
	}
	accountConfig, err := gatewaylogic.NewService(h.svc).AccountConfig(account)
	if err != nil {
		logger.WarnContext(runCtx, "读取网关渠道凭据失败",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("account_id", inbox.AccountID.String()),
			slog.String("provider", account.Provider),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return err
	}

	// typing 是尽力而为能力；不具备能力或平台调用失败时不影响 Agent 主流程。
	stopTyping := startTyping(runCtx, provider, accountConfig, event.Route)
	defer stopTyping()
	userMessage, err := h.svc.MessageRepo.FindByID(runCtx, inbox.UserID, *inbox.CoveMessageID)
	if err != nil {
		logger.WarnContext(runCtx, "读取网关用户消息失败",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("conversation_id", inbox.ConversationID.String()),
			slog.String("user_message_id", inbox.CoveMessageID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return err
	}

	// 仅把成功提取的附件文本重建为临时上下文，原始附件和外部文本不会写入日志。
	attachments := gatewayAttachments(userMessage.MetaData)
	var enableKnowledge *bool
	if inbox.ToolPolicy == models.ChannelToolPolicySafe {
		enabled := true
		enableKnowledge = &enabled
	}
	turnStartedAt := time.Now()
	logger.InfoContext(runCtx, "开始生成网关 Assistant 回复",
		slog.String("inbox_id", inbox.ID.String()),
		slog.String("account_id", inbox.AccountID.String()),
		slog.String("conversation_id", inbox.ConversationID.String()),
		slog.String("provider", account.Provider),
		slog.String("tool_policy", normalizeWorkerToolPolicy(inbox.ToolPolicy)),
		slog.Int("attachment_count", len(attachments)),
		slog.Bool("typing_enabled", provider.Descriptor().Capabilities.Typing),
	)
	result, err := chatlogic.RunTurn(runCtx, h.svc, chatlogic.TurnInput{
		UserID: inbox.UserID, ConversationID: *inbox.ConversationID, CurrentUserMessageID: *inbox.CoveMessageID,
		Message: userMessage.Content, Attachments: attachments, AgentConfigID: inbox.ResolvedAgentConfigID,
		EnableKnowledge: enableKnowledge, ToolPolicy: normalizeWorkerToolPolicy(inbox.ToolPolicy),
		AssistantMessageID: &assistantID, DiscardPartialOnError: true,
	}, nil)
	if err != nil {
		// 生成失败恢复为 queued，使任务重试仍能复用同一个确定性 Assistant ID。
		updateErr := h.svc.ChannelGatewayRepo.UpdateInboxEvent(ctx, inbox.ID, map[string]any{"status": models.ChannelInboxStatusQueued, "error_message": "Agent 回合执行失败"})
		logger.WarnContext(ctx, "网关 Assistant 回复生成失败，等待任务重试",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("conversation_id", inbox.ConversationID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
			slog.Int64("duration_ms", time.Since(turnStartedAt).Milliseconds()),
			slog.Bool("inbox_status_restored", updateErr == nil),
		)
		if updateErr != nil {
			logger.WarnContext(ctx, "恢复网关 Inbox 重试状态失败",
				slog.String("inbox_id", inbox.ID.String()),
				slog.String("error_kind", gatewayTurnErrorKind(updateErr)),
			)
		}
		return err
	}
	logger.InfoContext(runCtx, "网关 Assistant 回复生成完成",
		slog.String("inbox_id", inbox.ID.String()),
		slog.String("conversation_id", inbox.ConversationID.String()),
		slog.Int64("duration_ms", time.Since(turnStartedAt).Milliseconds()),
	)
	if err := h.finishTurn(runCtx, inbox, result.AssistantMessage); err != nil {
		return err
	}
	logger.InfoContext(runCtx, "网关 Agent 回合处理完成",
		slog.String("inbox_id", inbox.ID.String()),
		slog.String("conversation_id", inbox.ConversationID.String()),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
	return nil
}

// finishTurn 先建立幂等 Outbox，再将 Inbox 标记为 completed，最后投递异步发送任务。
func (h *GatewayTurnTask) finishTurn(ctx context.Context, inbox *models.ChannelInboxEvent, assistant *models.Message) error {
	logger := h.logger()
	if assistant == nil || inbox.ConversationID == nil {
		logger.WarnContext(ctx, "网关回合缺少 Assistant 结果",
			slog.String("inbox_id", inbox.ID.String()),
			slog.Bool("has_assistant", assistant != nil),
			slog.Bool("has_conversation", inbox.ConversationID != nil),
		)
		return errors.New("gateway assistant message is missing")
	}

	// InboxEventID 唯一约束保证恢复流程不会为同一回复创建多个 Outbox。
	outbox, err := h.svc.ChannelGatewayRepo.CreateOutboxMessage(ctx, &models.ChannelOutboxMessage{
		UserID: inbox.UserID, AccountID: inbox.AccountID, InboxEventID: inbox.ID,
		ConversationID: *inbox.ConversationID, AssistantMessageID: assistant.ID,
		DeliveryID: uuid.NewString(), Route: inboxRoute(inbox.NormalizedEvent), ReplyTo: inboxReply(inbox.NormalizedEvent), Content: assistant.Content,
		Status: models.ChannelOutboxStatusPending,
	})
	if err != nil {
		logger.WarnContext(ctx, "写入网关 Outbox 失败",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("assistant_message_id", assistant.ID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return err
	}
	logger.InfoContext(ctx, "网关最终回复已写入 Outbox",
		slog.String("inbox_id", inbox.ID.String()),
		slog.String("outbox_id", outbox.ID.String()),
		slog.String("assistant_message_id", assistant.ID.String()),
	)

	// 只有 Outbox 已持久化后才完成 Inbox，确保 completed 状态总有可恢复的投递记录。
	now := time.Now()
	if err := h.svc.ChannelGatewayRepo.UpdateInboxEvent(ctx, inbox.ID, map[string]any{
		"status": models.ChannelInboxStatusCompleted, "assistant_message_id": assistant.ID, "processed_at": &now, "error_message": "",
	}); err != nil {
		logger.WarnContext(ctx, "完成网关 Inbox 状态失败",
			slog.String("inbox_id", inbox.ID.String()),
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return err
	}
	logger.InfoContext(ctx, "网关 Inbox 已完成并等待最终投递",
		slog.String("inbox_id", inbox.ID.String()),
		slog.String("outbox_id", outbox.ID.String()),
	)
	return h.enqueueOutbox(ctx, outbox.ID)
}

// enqueueExistingOutbox 恢复 completed Inbox 对应的未终态 Outbox，终态消息不会重复发送。
func (h *GatewayTurnTask) enqueueExistingOutbox(ctx context.Context, inboxID uuid.UUID) error {
	logger := h.logger()
	outbox, err := h.svc.ChannelGatewayRepo.FindOutboxMessageByInboxEventID(ctx, inboxID)
	if err != nil {
		logger.WarnContext(ctx, "读取已完成回合的 Outbox 失败",
			slog.String("inbox_id", inboxID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return err
	}
	if outbox.Status == models.ChannelOutboxStatusSent || outbox.Status == models.ChannelOutboxStatusFailed || outbox.Status == models.ChannelOutboxStatusUnknown {
		logger.InfoContext(ctx, "网关 Outbox 已处于投递终态，跳过重复入队",
			slog.String("inbox_id", inboxID.String()),
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("outbox_status", outbox.Status),
		)
		return nil
	}
	logger.InfoContext(ctx, "恢复未完成的网关 Outbox 投递",
		slog.String("inbox_id", inboxID.String()),
		slog.String("outbox_id", outbox.ID.String()),
		slog.String("outbox_status", outbox.Status),
	)
	return h.enqueueOutbox(ctx, outbox.ID)
}

// enqueueOutbox 创建轻量投递任务；出站内容仍只保存在 Outbox，不进入队列 payload 和日志。
func (h *GatewayTurnTask) enqueueOutbox(ctx context.Context, outboxID uuid.UUID) error {
	logger := h.logger()
	task, err := types.NewGatewayDeliverTask(outboxID)
	if err != nil {
		logger.WarnContext(ctx, "创建网关 Outbox 投递任务失败",
			slog.String("outbox_id", outboxID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return err
	}
	info, err := h.svc.TaskProducer.Enqueue(ctx, task)
	if err != nil {
		logger.WarnContext(ctx, "网关 Outbox 投递任务入队失败，等待对账恢复",
			slog.String("outbox_id", outboxID.String()),
			slog.String("error_kind", gatewayTurnErrorKind(err)),
		)
		return err
	}
	logger.InfoContext(ctx, "网关 Outbox 投递任务已入队",
		slog.String("outbox_id", outboxID.String()),
		slog.String("task_id", info.ID),
	)
	return nil
}

// logger 返回任务专属 Logger；直接构造测试对象时回退到全局 Logger。
func (h *GatewayTurnTask) logger() *slog.Logger {
	if h != nil && h.log != nil {
		return h.log
	}
	return slog.Default()
}

// gatewayTurnErrorKind 返回可安全记录的错误分类，避免把渠道内容、Token 或媒体 URL 写入日志。
func gatewayTurnErrorKind(err error) string {
	if err == nil {
		return ""
	}
	return string(xerr.From(err).Kind)
}

func gatewayTurnPayload(task *types.Task) (*types.GatewayTurnPayload, error) {
	if task == nil {
		return nil, errors.New("gateway turn task is nil")
	}
	payload, ok := task.Payload.(*types.GatewayTurnPayload)
	if !ok || payload == nil || payload.InboxEventID == uuid.Nil {
		return nil, fmt.Errorf("invalid gateway turn payload %T", task.Payload)
	}
	return payload, nil
}

func gatewayAttachments(meta *models.MessageMetaData) []*types.MessageAttachment {
	if meta == nil {
		return nil
	}
	out := make([]*types.MessageAttachment, 0, len(meta.Attachments))
	for _, attachment := range meta.Attachments {
		if strings.TrimSpace(attachment.ExtractedText) == "" {
			continue
		}
		out = append(out, &types.MessageAttachment{FileName: attachment.FileName, Content: attachment.ExtractedText})
	}
	return out
}

func normalizeWorkerToolPolicy(value string) string {
	switch value {
	case models.ChannelToolPolicySafe:
		return flowchat.ToolPolicySafe
	case models.ChannelToolPolicyNone:
		return flowchat.ToolPolicyNone
	default:
		return flowchat.ToolPolicyInherit
	}
}

func inboxRoute(value models.JSONMap) models.JSONMap {
	route, _ := value["route"].(map[string]any)
	if route == nil {
		if typed, ok := value["route"].(models.JSONMap); ok {
			return typed
		}
		return models.JSONMap{}
	}
	return models.JSONMap(route)
}

func inboxReply(value models.JSONMap) models.JSONMap {
	messageID, _ := value["platform_message_id"].(string)
	if strings.TrimSpace(messageID) == "" {
		return nil
	}
	return models.JSONMap{"message_id": messageID}
}

// startTyping 启动输入指示；返回取消输入指示的函数。
func startTyping(ctx context.Context, sender corechannel.Provider, account corechannel.AccountConfig, route corechannel.Route) func() {
	if sender == nil || !sender.Descriptor().Capabilities.Typing {
		return func() {}
	}
	typingCtx, cancel := context.WithCancel(ctx)
	_ = sender.SetTyping(typingCtx, account, route, true)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				_ = sender.SetTyping(typingCtx, account, route, true)
			}
		}
	}()
	return func() {
		cancel()
		<-done
		_ = sender.SetTyping(context.Background(), account, route, false)
	}
}
