package modelconfig

import (
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

func configIDFromInput(input *request.UriConfigIDRequest) (uuid.UUID, error) {
	if input == nil {
		return uuid.Nil, xerr.BadRequest("模型配置 ID 无效")
	}
	id, err := uuid.Parse(input.ConfigID)
	if err != nil {
		return uuid.Nil, xerr.BadRequest("模型配置 ID 无效")
	}
	return id, nil
}
