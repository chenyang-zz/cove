package gateway

import (
	"context"
	"strings"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"log/slog"
)

// UpdateAccountLogic contains the updateAccount use case.
type UpdateAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewUpdateAccountLogic creates a UpdateAccountLogic.
func NewUpdateAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAccountLogic {
	return &UpdateAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.updateaccount"),
	}
}

// UpdateAccount 局部更新渠道账号并通知网关热重载
func (l *UpdateAccountLogic) UpdateAccount(userID uuid.UUID, input *request.UpdateChannelAccountDocRequest) (*response.ChannelAccountResponse, error) {
	if input == nil {
		return nil, xerr.BadRequest("渠道账号参数不能为空")
	}
	accountID, err := gatewayID(&input.UriGatewayIDRequest)
	if err != nil {
		return nil, err
	}
	current, err := l.svcCtx.ChannelGatewayRepo.FindAccountByID(l.ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	provider, ok := l.svcCtx.ChannelRegistry.Get(corechannel.ProviderName(current.Provider))
	if !ok {
		return nil, xerr.BadRequest("渠道 Provider 不可用")
	}
	values := make(map[string]any)
	if input.Name != nil {
		values["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Credentials != nil {
		if err := validateFields(provider.Descriptor().CredentialFields, input.Credentials, false); err != nil {
			return nil, err
		}
		encrypted := cloneSettings(current.EncryptedCredentials)
		for key, value := range input.Credentials {
			ciphertext, encryptErr := l.svcCtx.SecretCipher.Encrypt(value)
			if encryptErr != nil {
				return nil, xerr.Internal("渠道凭据加密失败", encryptErr)
			}
			encrypted[key] = ciphertext
		}
		values["encrypted_credentials"] = encrypted
	}
	if input.Settings != nil {
		if err := validateSettings(provider.Descriptor().SettingFields, input.Settings, true); err != nil {
			return nil, err
		}
		values["settings"] = cloneSettings(input.Settings)
	}
	if input.DefaultAgentConfigID != nil {
		if _, err := l.svcCtx.AgentConfigRepo.FindByID(l.ctx, userID, *input.DefaultAgentConfigID); err != nil {
			return nil, err
		}
		values["default_agent_config_id"] = *input.DefaultAgentConfigID
	} else if input.ClearDefaultAgent {
		values["default_agent_config_id"] = nil
	}
	if input.Enabled != nil {
		values["enabled"] = *input.Enabled
		if !*input.Enabled {
			values["status"] = models.ChannelAccountStatusDisabled
		}
	}
	if len(values) == 0 {
		return nil, xerr.BadRequest("更新字段不能为空")
	}
	row, err := l.svcCtx.ChannelGatewayRepo.UpdateAccount(l.ctx, userID, accountID, values)
	if err != nil {
		return nil, err
	}
	publishReload(l.ctx, l.svcCtx, accountID)
	return accountResponse(l.svcCtx, row)
}
