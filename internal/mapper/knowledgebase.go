package mapper

import (
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/transport/http/response"
)

func KnowledgeBaseToResponse(row *models.KnowledgeBase) *response.KnowledgeBaseResponse {
	if row == nil {
		return nil
	}
	return &response.KnowledgeBaseResponse{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Icon:        row.Icon,
		Color:       row.Color,
		IsDefault:   row.IsDefault,
		ChatEnabled: row.ChatEnabled,
		DocCount:    0,
		ImageCount:  0,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
