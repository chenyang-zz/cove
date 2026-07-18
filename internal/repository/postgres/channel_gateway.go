package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChannelGatewayRepository 使用 PostgreSQL 实现网关收件箱和发件箱。
type ChannelGatewayRepository struct{ db *gorm.DB }

// NewChannelGatewayRepository 创建渠道网关仓储。
func NewChannelGatewayRepository(db *gorm.DB) repository.ChannelGatewayRepository {
	return &ChannelGatewayRepository{db: db}
}

func (r *ChannelGatewayRepository) CreateAccount(ctx context.Context, userID uuid.UUID, row *models.ChannelAccount) (*models.ChannelAccount, error) {
	row.UserID = userID
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, xerr.Wrapf(err, "创建渠道账号失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) ListAccounts(ctx context.Context, userID uuid.UUID) ([]*models.ChannelAccount, error) {
	var rows []*models.ChannelAccount
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, xerr.Wrapf(err, "查询渠道账号失败")
	}
	return rows, nil
}

func (r *ChannelGatewayRepository) ListEnabledAccounts(ctx context.Context) ([]*models.ChannelAccount, error) {
	var rows []*models.ChannelAccount
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("id").Find(&rows).Error; err != nil {
		return nil, xerr.Wrapf(err, "查询已启用渠道账号失败")
	}
	return rows, nil
}

func (r *ChannelGatewayRepository) FindAccountByID(ctx context.Context, userID, accountID uuid.UUID) (*models.ChannelAccount, error) {
	row := &models.ChannelAccount{}
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", accountID, userID).First(row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NotFound("渠道账号不存在")
	}
	if err != nil {
		return nil, xerr.Wrapf(err, "查询渠道账号失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) FindAccountByPublicID(ctx context.Context, publicID string) (*models.ChannelAccount, error) {
	row := &models.ChannelAccount{}
	err := r.db.WithContext(ctx).Where("public_id = ? AND enabled = ?", publicID, true).First(row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NotFound("Webhook 不存在")
	}
	if err != nil {
		return nil, xerr.Wrapf(err, "查询 Webhook 账号失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) UpdateAccount(ctx context.Context, userID, accountID uuid.UUID, values map[string]any) (*models.ChannelAccount, error) {
	result := r.db.WithContext(ctx).Model(&models.ChannelAccount{}).
		Where("id = ? AND user_id = ?", accountID, userID).Updates(values)
	if result.Error != nil {
		return nil, xerr.Wrapf(result.Error, "更新渠道账号失败")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NotFound("渠道账号不存在")
	}
	return r.FindAccountByID(ctx, userID, accountID)
}

func (r *ChannelGatewayRepository) UpdateAccountHealth(ctx context.Context, accountID uuid.UUID, status, lastError string) error {
	now := time.Now()
	// 健康心跳不能修改 updated_at；Manager 用该字段判断账号配置是否需要重载。
	return r.db.WithContext(ctx).Model(&models.ChannelAccount{}).Where("id = ?", accountID).UpdateColumns(map[string]any{
		"status": status, "last_error": lastError, "last_seen_at": &now,
	}).Error
}

func (r *ChannelGatewayRepository) DeleteAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", accountID, userID).Delete(&models.ChannelAccount{})
	if result.Error != nil {
		return xerr.Wrapf(result.Error, "删除渠道账号失败")
	}
	if result.RowsAffected == 0 {
		return xerr.NotFound("渠道账号不存在")
	}
	return nil
}

func (r *ChannelGatewayRepository) ListPairings(ctx context.Context, userID, accountID uuid.UUID) ([]*models.ChannelIdentity, error) {
	var rows []*models.ChannelIdentity
	err := r.db.WithContext(ctx).Joins("JOIN channel_accounts ON channel_accounts.id = channel_identities.account_id").
		Where("channel_accounts.user_id = ? AND channel_identities.account_id = ?", userID, accountID).
		Order("channel_identities.updated_at DESC").Find(&rows).Error
	if err != nil {
		return nil, xerr.Wrapf(err, "查询配对请求失败")
	}
	return rows, nil
}

func (r *ChannelGatewayRepository) FindIdentity(ctx context.Context, accountID uuid.UUID, externalUserID, externalChatID string) (*models.ChannelIdentity, error) {
	row := &models.ChannelIdentity{}
	err := r.db.WithContext(ctx).Where("account_id = ? AND external_user_id = ? AND external_chat_id = ?", accountID, externalUserID, externalChatID).First(row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NotFound("渠道身份不存在")
	}
	if err != nil {
		return nil, xerr.Wrapf(err, "查询渠道身份失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) FindIdentityByID(ctx context.Context, userID, identityID uuid.UUID) (*models.ChannelIdentity, error) {
	row := &models.ChannelIdentity{}
	err := r.db.WithContext(ctx).Joins("JOIN channel_accounts ON channel_accounts.id = channel_identities.account_id").
		Where("channel_identities.id = ? AND channel_accounts.user_id = ?", identityID, userID).First(row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NotFound("配对请求不存在")
	}
	if err != nil {
		return nil, xerr.Wrapf(err, "查询配对请求失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) CountPendingIdentities(ctx context.Context, accountID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ChannelIdentity{}).
		Where("account_id = ? AND status = ? AND pairing_expires_at > ?", accountID, models.ChannelIdentityStatusPending, time.Now()).Count(&count).Error
	return count, err
}

func (r *ChannelGatewayRepository) CreateIdentity(ctx context.Context, row *models.ChannelIdentity) (*models.ChannelIdentity, error) {
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, xerr.Wrapf(err, "创建配对请求失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) UpdateIdentity(ctx context.Context, userID, identityID uuid.UUID, values map[string]any) (*models.ChannelIdentity, error) {
	var accountIDs []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&models.ChannelAccount{}).Where("user_id = ?", userID).Pluck("id", &accountIDs).Error; err != nil {
		return nil, err
	}
	result := r.db.WithContext(ctx).Model(&models.ChannelIdentity{}).Where("id = ? AND account_id IN ?", identityID, accountIDs).Updates(values)
	if result.Error != nil {
		return nil, xerr.Wrapf(result.Error, "更新配对请求失败")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NotFound("配对请求不存在")
	}
	return r.FindIdentityByID(ctx, userID, identityID)
}

func (r *ChannelGatewayRepository) ListBindings(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID) ([]*models.ChannelBinding, error) {
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if accountID != nil {
		query = query.Where("account_id = ?", *accountID)
	}
	var rows []*models.ChannelBinding
	if err := query.Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, xerr.Wrapf(err, "查询渠道绑定失败")
	}
	return rows, nil
}

func (r *ChannelGatewayRepository) FindBindingByID(ctx context.Context, userID, bindingID uuid.UUID) (*models.ChannelBinding, error) {
	row := &models.ChannelBinding{}
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", bindingID, userID).First(row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NotFound("渠道绑定不存在")
	}
	if err != nil {
		return nil, xerr.Wrapf(err, "查询渠道绑定失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) FindBindingByRoute(ctx context.Context, accountID uuid.UUID, routeKey string) (*models.ChannelBinding, error) {
	row := &models.ChannelBinding{}
	err := r.db.WithContext(ctx).Where("account_id = ? AND route_key = ?", accountID, routeKey).First(row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NotFound("渠道绑定不存在")
	}
	if err != nil {
		return nil, xerr.Wrapf(err, "查询渠道绑定失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) CreateBinding(ctx context.Context, userID uuid.UUID, row *models.ChannelBinding) (*models.ChannelBinding, error) {
	row.UserID = userID
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, xerr.Wrapf(err, "创建渠道绑定失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) UpdateBinding(ctx context.Context, userID, bindingID uuid.UUID, values map[string]any) (*models.ChannelBinding, error) {
	result := r.db.WithContext(ctx).Model(&models.ChannelBinding{}).Where("id = ? AND user_id = ?", bindingID, userID).Updates(values)
	if result.Error != nil {
		return nil, xerr.Wrapf(result.Error, "更新渠道绑定失败")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NotFound("渠道绑定不存在")
	}
	return r.FindBindingByID(ctx, userID, bindingID)
}

func (r *ChannelGatewayRepository) DeleteBinding(ctx context.Context, userID, bindingID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", bindingID, userID).Delete(&models.ChannelBinding{})
	if result.Error != nil {
		return xerr.Wrapf(result.Error, "删除渠道绑定失败")
	}
	if result.RowsAffected == 0 {
		return xerr.NotFound("渠道绑定不存在")
	}
	return nil
}

func (r *ChannelGatewayRepository) CreateInboxEvent(ctx context.Context, row *models.ChannelInboxEvent) (*models.ChannelInboxEvent, bool, error) {
	err := r.db.WithContext(ctx).Create(row).Error
	if err == nil {
		return row, true, nil
	}
	if !isUniqueViolation(err) {
		return nil, false, xerr.Wrapf(err, "持久化渠道事件失败")
	}
	existing := &models.ChannelInboxEvent{}
	if findErr := r.db.WithContext(ctx).Where("account_id = ? AND provider_event_id = ?", row.AccountID, row.ProviderEventID).First(existing).Error; findErr != nil {
		return nil, false, xerr.Wrapf(findErr, "查询重复渠道事件失败")
	}
	return existing, false, nil
}

func (r *ChannelGatewayRepository) FindInboxEventByID(ctx context.Context, id uuid.UUID) (*models.ChannelInboxEvent, error) {
	row := &models.ChannelInboxEvent{}
	if err := r.db.WithContext(ctx).First(row, "id = ?", id).Error; err != nil {
		return nil, xerr.Wrapf(err, "查询渠道事件失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) UpdateInboxEvent(ctx context.Context, id uuid.UUID, values map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.ChannelInboxEvent{}).Where("id = ?", id).Updates(values).Error
}

func (r *ChannelGatewayRepository) ListRecoverableInboxEvents(ctx context.Context, limit int) ([]*models.ChannelInboxEvent, error) {
	var rows []*models.ChannelInboxEvent
	err := r.db.WithContext(ctx).Where("status = ?", models.ChannelInboxStatusReceived).Order("created_at").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *ChannelGatewayRepository) CreateOutboxMessage(ctx context.Context, row *models.ChannelOutboxMessage) (*models.ChannelOutboxMessage, error) {
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		if isUniqueViolation(err) {
			return r.FindOutboxMessageByInboxEventID(ctx, row.InboxEventID)
		}
		return nil, xerr.Wrapf(err, "创建渠道出站消息失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) FindOutboxMessageByInboxEventID(ctx context.Context, inboxID uuid.UUID) (*models.ChannelOutboxMessage, error) {
	row := &models.ChannelOutboxMessage{}
	if err := r.db.WithContext(ctx).First(row, "inbox_event_id = ?", inboxID).Error; err != nil {
		return nil, xerr.Wrapf(err, "查询渠道事件出站消息失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) FindOutboxMessageByID(ctx context.Context, id uuid.UUID) (*models.ChannelOutboxMessage, error) {
	row := &models.ChannelOutboxMessage{}
	if err := r.db.WithContext(ctx).First(row, "id = ?", id).Error; err != nil {
		return nil, xerr.Wrapf(err, "查询渠道出站消息失败")
	}
	return row, nil
}

func (r *ChannelGatewayRepository) UpdateOutboxMessage(ctx context.Context, id uuid.UUID, values map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.ChannelOutboxMessage{}).Where("id = ?", id).Updates(values).Error
}

func (r *ChannelGatewayRepository) ListDeliverableOutboxMessages(ctx context.Context, limit int) ([]*models.ChannelOutboxMessage, error) {
	now := time.Now()
	var rows []*models.ChannelOutboxMessage
	err := r.db.WithContext(ctx).Where("status = ? OR (status = ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?))",
		models.ChannelOutboxStatusPending, models.ChannelOutboxStatusRetry, now).
		Order("created_at").Limit(limit).Find(&rows).Error
	return rows, err
}
