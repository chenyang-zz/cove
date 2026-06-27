package modelconfig

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/infrastructure/security"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

type SetModelDefaultLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

func NewSetModelDefaultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetModelDefaultLogic {
	return &SetModelDefaultLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.modelconfig.setmodeldefault"),
	}
}

func (l *SetModelDefaultLogic) SetModelDefault(userID uuid.UUID, input *request.UriConfigIDRequest) (*response.ModelResponse, error) {
	configID, err := configIDFromInput(input)
	if err != nil {
		return nil, err
	}
	config, err := l.svcCtx.ModelConfigRepo.FindByID(l.ctx, userID, configID)
	if err != nil {
		return nil, err
	}

	config.IsDefault = true
	config, err = l.svcCtx.ModelConfigRepo.Update(l.ctx, config)
	if err != nil {
		return nil, err
	}

	decodedApiKey, err := l.svcCtx.SecretCipher.Decrypt(config.APIKeyEncrypted)
	if err != nil {
		return nil, xerr.Internal("模型 API Key 解密失败", err)
	}

	return mapper.ModelConfigToResponse(config, security.MaskSecret(decodedApiKey)), nil
}
