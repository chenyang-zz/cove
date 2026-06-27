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

type CreateModelLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

func NewCreateModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateModelLogic {
	return &CreateModelLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.modelconfig.createmodel"),
	}
}

func (l *CreateModelLogic) CreateModel(userID uuid.UUID, input *request.CreateModelRequest) (*response.ModelResponse, error) {

	apiKeyEncrypted, err := l.svcCtx.SecretCipher.Encrypt(input.ApiKey)
	if err != nil {
		return nil, xerr.Internal("模型 API Key 加密失败", err)
	}

	row, err := l.svcCtx.ModelConfigRepo.Create(l.ctx, mapper.NewModelConfigFromCreate(userID, input, apiKeyEncrypted))
	if err != nil {
		return nil, err
	}
	return mapper.ModelConfigToResponse(row, security.MaskSecret(input.ApiKey)), nil
}
