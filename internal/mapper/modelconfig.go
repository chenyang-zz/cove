package mapper

import (
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/google/uuid"
)

func NewModelConfigFromCreate(userID uuid.UUID, input *request.CreateModelRequest, encryptedAPIKey string) *models.ModelConfig {
	return &models.ModelConfig{
		UserID:          userID,
		Type:            input.Type,
		Provider:        input.Provider,
		Name:            input.Name,
		ModelName:       input.ModelName,
		APIKeyEncrypted: encryptedAPIKey,
		BaseURL:         input.BaseUrl,
		Capability:      models.StringList(input.Capability),
		IsDefault:       input.IsDefault,
	}
}

func ApplyUpdateModelConfig(row *models.ModelConfig, input *request.UpdateModelRequest) {
	if row == nil || input == nil {
		return
	}
	util.AssignIfNotEmpty(&row.Name, input.Name)
	util.AssignIfNotEmpty(&row.ModelName, input.ModelName)
	util.AssignIfNotEmpty(&row.BaseURL, input.BaseUrl)
	if input.Capability != nil {
		row.Capability = models.StringList(input.Capability)
	}
}

func ModelConfigToResponse(row *models.ModelConfig, apiKeyMasked string) *response.ModelResponse {
	if row == nil {
		return nil
	}
	return &response.ModelResponse{
		ID:           row.ID,
		Type:         row.Type,
		Provider:     row.Provider,
		Name:         row.Name,
		ModelName:    row.ModelName,
		APIKeyMasked: apiKeyMasked,
		BaseURL:      row.BaseURL,
		Capability:   []string(row.Capability),
		IsDefault:    row.IsDefault,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func ModelConfigsToListResponse(rows []*models.ModelConfig, mask func(*models.ModelConfig) (string, bool)) *response.ListModelsResponse {
	out := make([]*response.ModelResponse, 0, len(rows))
	for _, row := range rows {
		masked, ok := mask(row)
		if !ok {
			continue
		}
		out = append(out, ModelConfigToResponse(row, masked))
	}
	return &response.ListModelsResponse{List: out}
}
