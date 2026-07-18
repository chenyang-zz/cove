// Package runtime 管理多实例网关账号租约、动态重载和可靠性对账。
package runtime

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/infrastructure/lease"
	gatewaylogic "github.com/boxify/api-go/internal/logic/gateway"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
)

const reloadChannel = "gateway:reload"

type runningAccount struct {
	version int64
	cancel  context.CancelFunc
}

// Manager 对账数据库账号并保证每个 Receiver 只有一个实例运行。
type Manager struct {
	svc     *svc.ServiceContext
	service *gatewaylogic.Service
	log     *slog.Logger
	mu      sync.Mutex
	running map[string]runningAccount
}

// NewManager 创建网关运行管理器。
func NewManager(svcCtx *svc.ServiceContext) *Manager {
	return &Manager{svc: svcCtx, service: gatewaylogic.NewService(svcCtx), log: xlog.Component("gateway.runtime"), running: make(map[string]runningAccount)}
}

// Run 监听重载通知并周期性对账账号、Inbox 和 Outbox。
func (m *Manager) Run(ctx context.Context) error {
	interval := parsePositiveDuration(m.svc.Config.Gateway.ReconcileInterval, 30*time.Second)
	pubsub := m.svc.Redis.Raw().Subscribe(ctx, reloadChannel)
	defer pubsub.Close()
	if err := m.reconcile(ctx); err != nil {
		m.log.WarnContext(ctx, "网关首次对账失败", slog.String("error", "reconcile failed"))
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			m.stopAll()
			return nil
		case <-ticker.C:
			_ = m.reconcile(ctx)
		case _, ok := <-pubsub.Channel():
			if !ok {
				m.stopAll()
				return errors.New("gateway reload subscription closed")
			}
			_ = m.reconcile(ctx)
		}
	}
}

func (m *Manager) reconcile(ctx context.Context) error {
	accounts, err := m.svc.ChannelGatewayRepo.ListEnabledAccounts(ctx)
	if err != nil {
		return err
	}
	wanted := make(map[string]*models.ChannelAccount, len(accounts))
	for _, account := range accounts {
		if account == nil || account.Provider == string(corechannel.ProviderWebhook) {
			continue
		}
		wanted[account.ID.String()] = account
	}
	m.mu.Lock()
	for id, running := range m.running {
		account, ok := wanted[id]
		if !ok || running.version != account.UpdatedAt.UnixNano() {
			running.cancel()
			delete(m.running, id)
		}
	}
	for id, account := range wanted {
		if _, ok := m.running[id]; ok {
			continue
		}
		accountCtx, cancel := context.WithCancel(ctx)
		version := account.UpdatedAt.UnixNano()
		m.running[id] = runningAccount{version: version, cancel: cancel}
		go m.runAccount(accountCtx, account, version)
	}
	m.mu.Unlock()
	if err := m.service.RecoverInbox(ctx, 100); err != nil {
		m.log.WarnContext(ctx, "网关 Inbox 对账失败", slog.String("error", "inbox reconcile failed"))
	}
	m.recoverOutbox(ctx)
	return nil
}

func (m *Manager) runAccount(ctx context.Context, account *models.ChannelAccount, version int64) {
	defer m.removeRunning(account.ID.String(), version)
	ttl := parsePositiveDuration(m.svc.Config.Gateway.LeaseTTL, 45*time.Second)
	accountLease, acquired, err := lease.Acquire(ctx, m.svc.Redis.Raw(), "gateway:receiver:"+account.ID.String(), ttl)
	if err != nil || !acquired {
		return
	}
	leaseCtx, stopKeepAlive := accountLease.KeepAlive(ctx)
	defer func() {
		stopKeepAlive()
		_ = accountLease.Release(context.Background())
	}()
	provider, ok := m.svc.ChannelRegistry.Get(corechannel.ProviderName(account.Provider))
	if !ok {
		_ = m.svc.ChannelGatewayRepo.UpdateAccountHealth(ctx, account.ID, models.ChannelAccountStatusDegraded, "Provider 不可用")
		return
	}
	accountConfig, err := m.service.AccountConfig(account)
	if err != nil {
		_ = m.svc.ChannelGatewayRepo.UpdateAccountHealth(ctx, account.ID, models.ChannelAccountStatusDegraded, "凭据解密失败")
		return
	}
	handler := corechannel.EventHandlerFunc(func(eventCtx context.Context, event corechannel.InboundEvent) error {
		_, _, handleErr := m.service.HandleInbound(eventCtx, account, event)
		return handleErr
	})
	backoff := time.Second
	for leaseCtx.Err() == nil {
		_ = m.svc.ChannelGatewayRepo.UpdateAccountHealth(leaseCtx, account.ID, models.ChannelAccountStatusHealthy, "")
		err := provider.Receive(leaseCtx, accountConfig, handler)
		if leaseCtx.Err() != nil {
			return
		}
		_ = m.svc.ChannelGatewayRepo.UpdateAccountHealth(ctx, account.ID, models.ChannelAccountStatusDegraded, "渠道连接中断，正在重连")
		m.log.WarnContext(ctx, "渠道 Receiver 中断",
			slog.String("account_id", account.ID.String()), slog.String("provider", account.Provider), slog.String("error", safeReceiverError(err)),
		)
		select {
		case <-leaseCtx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

func (m *Manager) recoverOutbox(ctx context.Context) {
	rows, err := m.svc.ChannelGatewayRepo.ListDeliverableOutboxMessages(ctx, 100)
	if err != nil {
		return
	}
	for _, row := range rows {
		task, taskErr := types.NewGatewayDeliverTask(row.ID)
		if taskErr != nil {
			continue
		}
		_, _ = m.svc.TaskProducer.Enqueue(ctx, task)
	}
}

func (m *Manager) removeRunning(id string, version int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if running, ok := m.running[id]; ok && running.version == version {
		delete(m.running, id)
	}
}

func (m *Manager) stopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, running := range m.running {
		running.cancel()
		delete(m.running, id)
	}
}

func parsePositiveDuration(value string, fallback time.Duration) time.Duration {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return fallback
	}
	return duration
}

func safeReceiverError(err error) string {
	if err == nil {
		return "connection closed"
	}
	return "connection failed"
}
