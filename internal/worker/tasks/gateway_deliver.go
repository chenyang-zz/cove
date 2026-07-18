package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/infrastructure/lease"
	gatewaylogic "github.com/boxify/api-go/internal/logic/gateway"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// GatewayDeliverTask 按回执状态可靠投递最终文本。
type GatewayDeliverTask struct {
	svc *svc.ServiceContext
	log *slog.Logger
}

// NewGatewayDeliverTask 创建发件箱 Worker。
func NewGatewayDeliverTask(svcCtx *svc.ServiceContext) *GatewayDeliverTask {
	return &GatewayDeliverTask{svc: svcCtx, log: xlog.Component("worker.gateway.deliver")}
}

// Handle 区分暂时、永久和不确定错误，禁止对不确定结果盲目重发。
func (h *GatewayDeliverTask) Handle(ctx context.Context, task *types.Task) error {
	startedAt := time.Now()
	logger := h.logger()

	// 队列只传递稳定 Outbox ID，消息正文和渠道路由始终从数据库快照读取。
	payload, err := gatewayDeliverPayload(task)
	if err != nil {
		logger.WarnContext(ctx, "网关 Outbox 投递任务载荷无效",
			slog.String("error_kind", gatewayDeliverErrorKind(err)),
		)
		return skipRetry(err)
	}
	logger.InfoContext(ctx, "开始执行网关 Outbox 投递",
		slog.String("outbox_id", payload.OutboxMessageID.String()),
	)

	outbox, err := h.svc.ChannelGatewayRepo.FindOutboxMessageByID(ctx, payload.OutboxMessageID)
	if err != nil {
		logger.WarnContext(ctx, "读取网关 Outbox 失败，停止投递",
			slog.String("outbox_id", payload.OutboxMessageID.String()),
			slog.String("error_kind", gatewayDeliverErrorKind(err)),
		)
		return skipRetry(err)
	}

	// 终态消息必须幂等退出；sending 表示上次进程可能已发出请求，结果不可安全推断。
	switch outbox.Status {
	case models.ChannelOutboxStatusSent, models.ChannelOutboxStatusFailed, models.ChannelOutboxStatusUnknown:
		logger.InfoContext(ctx, "网关 Outbox 已处于投递终态，跳过重复发送",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("account_id", outbox.AccountID.String()),
			slog.String("outbox_status", outbox.Status),
			slog.Int("attempt", outbox.AttemptCount),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return nil
	case models.ChannelOutboxStatusSending:
		updateErr := h.svc.ChannelGatewayRepo.UpdateOutboxMessage(ctx, outbox.ID, map[string]any{"status": models.ChannelOutboxStatusUnknown, "last_error": "上次发送结果不确定，已停止自动重发"})
		logger.WarnContext(ctx, "网关 Outbox 遗留在 sending 状态，停止自动重发",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("account_id", outbox.AccountID.String()),
			slog.Int("attempt", outbox.AttemptCount),
			slog.Bool("unknown_status_persisted", updateErr == nil),
		)
		if updateErr != nil {
			logger.WarnContext(ctx, "持久化网关 Outbox unknown 状态失败",
				slog.String("outbox_id", outbox.ID.String()),
				slog.String("error_kind", gatewayDeliverErrorKind(updateErr)),
			)
		}
		return skipRetry(errors.New("outbox was left in sending state"))
	}

	// 单个 Outbox 使用短租约锁串行发送，避免多个 Worker 同时调用渠道 API。
	lock, acquired, err := lease.Acquire(ctx, h.svc.Redis.Raw(), "gateway:deliver:lock:"+outbox.ID.String(), time.Minute)
	if err != nil {
		logger.WarnContext(ctx, "获取网关 Outbox 投递锁失败",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("error_kind", gatewayDeliverErrorKind(err)),
		)
		return err
	}
	if !acquired {
		logger.InfoContext(ctx, "网关 Outbox 正由其他 Worker 投递，等待任务重试",
			slog.String("outbox_id", outbox.ID.String()),
		)
		return errors.New("gateway outbox is busy")
	}
	logger.InfoContext(ctx, "网关 Outbox 投递锁已获取",
		slog.String("outbox_id", outbox.ID.String()),
	)
	defer func() {
		if releaseErr := lock.Release(context.Background()); releaseErr != nil {
			logger.Warn("释放网关 Outbox 投递锁失败，等待租约自动过期",
				slog.String("outbox_id", outbox.ID.String()),
				slog.String("error_kind", gatewayDeliverErrorKind(releaseErr)),
			)
		}
	}()

	// 通过 Outbox 的租户键重新加载 Account，禁止使用队列 payload 绕过账号归属校验。
	account, err := h.svc.ChannelGatewayRepo.FindAccountByID(ctx, outbox.UserID, outbox.AccountID)
	if err != nil || !account.Enabled {
		updateErr := h.svc.ChannelGatewayRepo.UpdateOutboxMessage(ctx, outbox.ID, map[string]any{"status": models.ChannelOutboxStatusFailed, "last_error": "渠道账号不可用"})
		attrs := []any{
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("account_id", outbox.AccountID.String()),
			slog.Bool("account_loaded", err == nil),
			slog.Bool("failed_status_persisted", updateErr == nil),
		}
		if err != nil {
			attrs = append(attrs, slog.String("error_kind", gatewayDeliverErrorKind(err)))
		}
		logger.WarnContext(ctx, "网关渠道账号不可用，停止 Outbox 投递", attrs...)
		if updateErr != nil {
			logger.WarnContext(ctx, "持久化网关 Outbox failed 状态失败",
				slog.String("outbox_id", outbox.ID.String()),
				slog.String("error_kind", gatewayDeliverErrorKind(updateErr)),
			)
		}
		return skipRetry(errors.New("gateway account is unavailable"))
	}
	provider, ok := h.svc.ChannelRegistry.Get(corechannel.ProviderName(account.Provider))
	if !ok {
		logger.WarnContext(ctx, "网关渠道 Provider 不可用",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("account_id", outbox.AccountID.String()),
			slog.String("provider", account.Provider),
		)
		return skipRetry(errors.New("gateway provider is unavailable"))
	}
	accountConfig, err := gatewaylogic.NewService(h.svc).AccountConfig(account)
	if err != nil {
		logger.WarnContext(ctx, "读取网关渠道凭据失败",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("account_id", outbox.AccountID.String()),
			slog.String("provider", account.Provider),
			slog.String("error_kind", gatewayDeliverErrorKind(err)),
		)
		return err
	}

	// 路由和回复引用在发送前重新校验；日志不记录外部 chat/thread/message ID。
	route, err := decodeOutboxRoute(outbox.Route)
	if err != nil {
		logger.WarnContext(ctx, "解析网关 Outbox 路由失败",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("account_id", outbox.AccountID.String()),
			slog.String("error_kind", gatewayDeliverErrorKind(err)),
		)
		return skipRetry(err)
	}
	replyTo, err := decodeOutboxReply(outbox.ReplyTo)
	if err != nil {
		logger.WarnContext(ctx, "解析网关 Outbox 回复引用失败",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("error_kind", gatewayDeliverErrorKind(err)),
		)
		return skipRetry(err)
	}

	// 先写 sending 和递增 attempt，再调用 Provider；进程在调用期间退出会在下次被判定为 unknown。
	attempt := outbox.AttemptCount + 1
	if err := h.svc.ChannelGatewayRepo.UpdateOutboxMessage(ctx, outbox.ID, map[string]any{"status": models.ChannelOutboxStatusSending, "attempt_count": attempt}); err != nil {
		logger.WarnContext(ctx, "更新网关 Outbox sending 状态失败",
			slog.String("outbox_id", outbox.ID.String()),
			slog.Int("attempt", attempt),
			slog.String("error_kind", gatewayDeliverErrorKind(err)),
		)
		return err
	}
	logger.InfoContext(ctx, "开始调用渠道 Provider 投递最终回复",
		slog.String("outbox_id", outbox.ID.String()),
		slog.String("account_id", outbox.AccountID.String()),
		slog.String("delivery_id", outbox.DeliveryID),
		slog.String("provider", account.Provider),
		slog.Int("attempt", attempt),
		slog.Bool("has_reply_reference", replyTo != nil),
	)
	sendStartedAt := time.Now()
	receipt, sendErr := provider.Send(ctx, accountConfig, corechannel.OutboundMessage{
		DeliveryID: outbox.DeliveryID, Route: route, Text: outbox.Content, ReplyTo: replyTo,
	})
	sendDuration := time.Since(sendStartedAt)
	receiptJSON := receiptMap(receipt)

	// Receipt.State 是唯一的重试判据；底层 error 只用于返回队列框架，不推断是否已发送。
	switch receipt.State {
	case corechannel.DeliverySent:
		now := time.Now()
		if err := h.svc.ChannelGatewayRepo.UpdateOutboxMessage(ctx, outbox.ID, map[string]any{
			"status": models.ChannelOutboxStatusSent, "platform_message_id": receipt.PlatformMessageID,
			"receipt": receiptJSON, "last_error": "", "sent_at": &now, "next_attempt_at": nil,
		}); err != nil {
			logger.WarnContext(ctx, "持久化网关 Outbox sent 回执失败",
				slog.String("outbox_id", outbox.ID.String()),
				slog.String("provider", account.Provider),
				slog.Int("attempt", attempt),
				slog.String("error_kind", gatewayDeliverErrorKind(err)),
			)
			return err
		}
		logger.InfoContext(ctx, "网关 Outbox 投递成功",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("account_id", outbox.AccountID.String()),
			slog.String("provider", account.Provider),
			slog.String("delivery_state", string(receipt.State)),
			slog.Int("attempt", attempt),
			slog.Int64("send_duration_ms", sendDuration.Milliseconds()),
			slog.Bool("provider_returned_error", sendErr != nil),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return nil
	case corechannel.DeliveryPermanent:
		updateErr := h.svc.ChannelGatewayRepo.UpdateOutboxMessage(ctx, outbox.ID, map[string]any{"status": models.ChannelOutboxStatusFailed, "receipt": receiptJSON, "last_error": safeDeliveryError(receipt)})
		logger.WarnContext(ctx, "网关 Outbox 遇到永久错误，停止自动重试",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("account_id", outbox.AccountID.String()),
			slog.String("provider", account.Provider),
			slog.String("delivery_state", string(receipt.State)),
			slog.Int("attempt", attempt),
			slog.Int64("send_duration_ms", sendDuration.Milliseconds()),
			slog.String("send_error_kind", gatewayDeliverErrorKind(sendErr)),
			slog.Bool("failed_status_persisted", updateErr == nil),
			slog.Bool("has_provider_error_code", receipt.ErrorCode != ""),
		)
		if updateErr != nil {
			logger.WarnContext(ctx, "持久化网关 Outbox failed 回执失败",
				slog.String("outbox_id", outbox.ID.String()),
				slog.String("error_kind", gatewayDeliverErrorKind(updateErr)),
			)
		}
		return skipRetry(firstDeliveryError(sendErr))
	case corechannel.DeliveryUnknown:
		updateErr := h.svc.ChannelGatewayRepo.UpdateOutboxMessage(ctx, outbox.ID, map[string]any{"status": models.ChannelOutboxStatusUnknown, "receipt": receiptJSON, "last_error": safeDeliveryError(receipt)})
		logger.WarnContext(ctx, "网关 Outbox 发送结果不确定，禁止盲目重发",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("account_id", outbox.AccountID.String()),
			slog.String("provider", account.Provider),
			slog.String("delivery_state", string(receipt.State)),
			slog.Int("attempt", attempt),
			slog.Int64("send_duration_ms", sendDuration.Milliseconds()),
			slog.String("send_error_kind", gatewayDeliverErrorKind(sendErr)),
			slog.Bool("unknown_status_persisted", updateErr == nil),
			slog.Bool("has_provider_error_code", receipt.ErrorCode != ""),
		)
		if updateErr != nil {
			logger.WarnContext(ctx, "持久化网关 Outbox unknown 回执失败",
				slog.String("outbox_id", outbox.ID.String()),
				slog.String("error_kind", gatewayDeliverErrorKind(updateErr)),
			)
		}
		return skipRetry(firstDeliveryError(sendErr))
	default:
		delay := retryDelay(attempt)
		nextAttempt := time.Now().Add(delay)
		updateErr := h.svc.ChannelGatewayRepo.UpdateOutboxMessage(ctx, outbox.ID, map[string]any{
			"status": models.ChannelOutboxStatusRetry, "receipt": receiptJSON,
			"last_error": safeDeliveryError(receipt), "next_attempt_at": &nextAttempt,
		})
		logger.WarnContext(ctx, "网关 Outbox 暂时投递失败，等待退避重试",
			slog.String("outbox_id", outbox.ID.String()),
			slog.String("account_id", outbox.AccountID.String()),
			slog.String("provider", account.Provider),
			slog.String("delivery_state", string(receipt.State)),
			slog.Int("attempt", attempt),
			slog.Int64("send_duration_ms", sendDuration.Milliseconds()),
			slog.String("send_error_kind", gatewayDeliverErrorKind(sendErr)),
			slog.Int64("retry_delay_ms", delay.Milliseconds()),
			slog.Bool("retry_status_persisted", updateErr == nil),
			slog.Bool("has_provider_error_code", receipt.ErrorCode != ""),
		)
		if updateErr != nil {
			logger.WarnContext(ctx, "持久化网关 Outbox retry 回执失败",
				slog.String("outbox_id", outbox.ID.String()),
				slog.String("error_kind", gatewayDeliverErrorKind(updateErr)),
			)
		}
		return firstDeliveryError(sendErr)
	}
}

// logger 返回任务专属 Logger；直接构造测试对象时回退到全局 Logger。
func (h *GatewayDeliverTask) logger() *slog.Logger {
	if h != nil && h.log != nil {
		return h.log
	}
	return slog.Default()
}

// gatewayDeliverErrorKind 返回可安全记录的错误分类，避免泄露正文、Token、回调 URL 或平台响应。
func gatewayDeliverErrorKind(err error) string {
	if err == nil {
		return ""
	}
	return string(xerr.From(err).Kind)
}

// gatewayDeliverPayload 校验任务 payload，并返回唯一的 Outbox 定位信息。
func gatewayDeliverPayload(task *types.Task) (*types.GatewayDeliverPayload, error) {
	if task == nil {
		return nil, errors.New("gateway deliver task is nil")
	}
	payload, ok := task.Payload.(*types.GatewayDeliverPayload)
	if !ok || payload == nil || payload.OutboxMessageID == uuid.Nil {
		return nil, fmt.Errorf("invalid gateway deliver payload %T", task.Payload)
	}
	return payload, nil
}

// decodeOutboxRoute 从持久化快照还原渠道路由，并拒绝缺少账号或聊天标识的消息。
func decodeOutboxRoute(value models.JSONMap) (corechannel.Route, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return corechannel.Route{}, err
	}
	var route corechannel.Route
	if err := json.Unmarshal(data, &route); err != nil {
		return corechannel.Route{}, err
	}
	if route.AccountID == "" || route.ChatID == "" {
		return corechannel.Route{}, errors.New("outbox route is incomplete")
	}
	return route, nil
}

// decodeOutboxReply 从持久化快照还原可选回复引用；空值会降级为普通发送。
func decodeOutboxReply(value models.JSONMap) (*corechannel.ReplyReference, error) {
	if len(value) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var reply corechannel.ReplyReference
	if err := json.Unmarshal(data, &reply); err != nil {
		return nil, err
	}
	if reply.MessageID == "" {
		return nil, errors.New("outbox reply reference is incomplete")
	}
	return &reply, nil
}

// receiptMap 将 Provider 回执转换为可持久化字段，不包含发送正文和账号凭据。
func receiptMap(receipt corechannel.Receipt) models.JSONMap {
	return models.JSONMap{
		"delivery_id": receipt.DeliveryID, "state": string(receipt.State),
		"platform_message_id": receipt.PlatformMessageID, "error_code": receipt.ErrorCode,
	}
}

// retryDelay 返回指数退避时间，并将指数限制在第六次尝试以避免无限增长。
func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(1<<(attempt-1)) * 5 * time.Second
}

// safeDeliveryError 返回可持久化的渠道错误摘要，不使用底层 error 文本。
func safeDeliveryError(receipt corechannel.Receipt) string {
	if receipt.ErrorCode != "" {
		return "渠道发送失败（" + receipt.ErrorCode + "）"
	}
	return "渠道发送失败"
}

// firstDeliveryError 保留 Provider 返回的错误；缺失时生成队列重试所需的通用错误。
func firstDeliveryError(err error) error {
	if err != nil {
		return err
	}
	return errors.New("channel delivery failed")
}
