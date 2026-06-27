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

type UpdateModelLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

func NewUpdateModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateModelLogic {
	return &UpdateModelLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.modelconfig.updatemodel"),
	}
}

func (l *UpdateModelLogic) UpdateModel(userID uuid.UUID, input *request.UpdateModelRequest) (*response.ModelResponse, error) {
	configID, err := configIDFromInput(&input.UriConfigIDRequest)
	if err != nil {
		return nil, err
	}
	modelConfig, err := l.svcCtx.ModelConfigRepo.FindByID(l.ctx, userID, configID)
	if err != nil {
		return nil, err
	}

	mapper.ApplyUpdateModelConfig(modelConfig, input)
	if input.ApiKey != nil && *input.ApiKey != "" {
		apiKeyEncrypted, err := l.svcCtx.SecretCipher.Encrypt(*input.ApiKey)
		if err != nil {
			return nil, xerr.Internal("模型 API Key 加密失败", err)
		}
		modelConfig.APIKeyEncrypted = apiKeyEncrypted
	}

	modelConfig, err = l.svcCtx.ModelConfigRepo.Update(l.ctx, modelConfig)
	if err != nil {
		return nil, err
	}

	decodedApiKey, err := l.svcCtx.SecretCipher.Decrypt(modelConfig.APIKeyEncrypted)
	if err != nil {
		return nil, xerr.Internal("模型 API Key 解密失败", err)
	}
	return mapper.ModelConfigToResponse(modelConfig, security.MaskSecret(decodedApiKey)), nil
}
