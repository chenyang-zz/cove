package repository

import (
	"context"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/models"
	"github.com/google/uuid"
)

type ModelConfigRepository interface {
	Create(ctx context.Context, modelConfig *models.ModelConfig) (*models.ModelConfig, error)
	Update(ctx context.Context, modelConfig *models.ModelConfig) (*models.ModelConfig, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	List(ctx context.Context, userID uuid.UUID, modelType *domain.ModelType) ([]*models.ModelConfig, error)
	FindByID(ctx context.Context, userID uuid.UUID, configID uuid.UUID) (*models.ModelConfig, error)
}
