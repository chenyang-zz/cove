package repository

import (
	"context"

	"github.com/boxify/api-go/internal/models"
	"github.com/google/uuid"
)

// ChannelGatewayRepository 汇总网关控制面和可靠数据面所需的持久化操作。
type ChannelGatewayRepository interface {
	CreateAccount(context.Context, uuid.UUID, *models.ChannelAccount) (*models.ChannelAccount, error)
	ListAccounts(context.Context, uuid.UUID) ([]*models.ChannelAccount, error)
	ListEnabledAccounts(context.Context) ([]*models.ChannelAccount, error)
	FindAccountByID(context.Context, uuid.UUID, uuid.UUID) (*models.ChannelAccount, error)
	FindAccountByPublicID(context.Context, string) (*models.ChannelAccount, error)
	UpdateAccount(context.Context, uuid.UUID, uuid.UUID, map[string]any) (*models.ChannelAccount, error)
	UpdateAccountHealth(context.Context, uuid.UUID, string, string) error
	DeleteAccount(context.Context, uuid.UUID, uuid.UUID) error

	ListPairings(context.Context, uuid.UUID, uuid.UUID) ([]*models.ChannelIdentity, error)
	FindIdentity(context.Context, uuid.UUID, string, string) (*models.ChannelIdentity, error)
	FindIdentityByID(context.Context, uuid.UUID, uuid.UUID) (*models.ChannelIdentity, error)
	CountPendingIdentities(context.Context, uuid.UUID) (int64, error)
	CreateIdentity(context.Context, *models.ChannelIdentity) (*models.ChannelIdentity, error)
	UpdateIdentity(context.Context, uuid.UUID, uuid.UUID, map[string]any) (*models.ChannelIdentity, error)

	ListBindings(context.Context, uuid.UUID, *uuid.UUID) ([]*models.ChannelBinding, error)
	FindBindingByID(context.Context, uuid.UUID, uuid.UUID) (*models.ChannelBinding, error)
	FindBindingByRoute(context.Context, uuid.UUID, string) (*models.ChannelBinding, error)
	CreateBinding(context.Context, uuid.UUID, *models.ChannelBinding) (*models.ChannelBinding, error)
	UpdateBinding(context.Context, uuid.UUID, uuid.UUID, map[string]any) (*models.ChannelBinding, error)
	DeleteBinding(context.Context, uuid.UUID, uuid.UUID) error

	CreateInboxEvent(context.Context, *models.ChannelInboxEvent) (*models.ChannelInboxEvent, bool, error)
	FindInboxEventByID(context.Context, uuid.UUID) (*models.ChannelInboxEvent, error)
	UpdateInboxEvent(context.Context, uuid.UUID, map[string]any) error
	ListRecoverableInboxEvents(context.Context, int) ([]*models.ChannelInboxEvent, error)

	CreateOutboxMessage(context.Context, *models.ChannelOutboxMessage) (*models.ChannelOutboxMessage, error)
	FindOutboxMessageByID(context.Context, uuid.UUID) (*models.ChannelOutboxMessage, error)
	FindOutboxMessageByInboxEventID(context.Context, uuid.UUID) (*models.ChannelOutboxMessage, error)
	UpdateOutboxMessage(context.Context, uuid.UUID, map[string]any) error
	ListDeliverableOutboxMessages(context.Context, int) ([]*models.ChannelOutboxMessage, error)
}
