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

// CreateAccountLogic contains the createAccount use case.
type CreateAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewCreateAccountLogic creates a CreateAccountLogic.
func NewCreateAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAccountLogic {
	return &CreateAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.gateway.createaccount"),
	}
}

// CreateAccount 创建渠道账号并加密保存凭据
func (l *CreateAccountLogic) CreateAccount(userID uuid.UUID, input *request.CreateChannelAccountRequest) (*response.ChannelAccountResponse, error) {
	if input == nil {
		return nil, xerr.BadRequest("渠道账号参数不能为空")
	}
	provider, ok := l.svcCtx.ChannelRegistry.Get(corechannel.ProviderName(input.Provider))
	if !ok {
		return nil, xerr.BadRequest("不支持的渠道 Provider")
	}
	if err := validateFields(provider.Descriptor().CredentialFields, input.Credentials, true); err != nil {
		return nil, err
	}
	if err := validateSettings(provider.Descriptor().SettingFields, input.Settings, true); err != nil {
		return nil, err
	}
	if input.DefaultAgentConfigID != nil {
		if _, err := l.svcCtx.AgentConfigRepo.FindByID(l.ctx, userID, *input.DefaultAgentConfigID); err != nil {
			return nil, err
		}
	}
	encrypted, err := encryptCredentials(l.svcCtx.SecretCipher, input.Credentials)
	if err != nil {
		return nil, err
	}
	publicID, err := randomPublicID()
	if err != nil {
		return nil, xerr.Internal("生成渠道公开 ID 失败", err)
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	row, err := l.svcCtx.ChannelGatewayRepo.CreateAccount(l.ctx, userID, &models.ChannelAccount{
		Provider: input.Provider, Name: strings.TrimSpace(input.Name), PublicID: publicID,
		EncryptedCredentials: encrypted, Settings: cloneSettings(input.Settings),
		DefaultAgentConfigID: input.DefaultAgentConfigID, Enabled: enabled, Status: models.ChannelAccountStatusUnknown,
	})
	if err != nil {
		return nil, err
	}
	publishReload(l.ctx, l.svcCtx, row.ID)
	return accountResponse(l.svcCtx, row)
}
