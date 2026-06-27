package postgres

import (
	"context"
	"errors"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ModelConfigRepository struct {
	db *gorm.DB
}

func NewModelConfigRepository(db *gorm.DB) repository.ModelConfigRepository {
	return &ModelConfigRepository{db: db}
}

func (r *ModelConfigRepository) Create(ctx context.Context, modelConfig *models.ModelConfig) (*models.ModelConfig, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if modelConfig.IsDefault {
			if err := tx.Model(&models.ModelConfig{}).
				Where("user_id = ? AND type = ?", modelConfig.UserID, modelConfig.Type).
				Update("is_default", false).Error; err != nil {
				return xerr.Wrapf(err, "更新默认模型配置失败")
			}
		}
		if err := tx.Create(&modelConfig).Error; err != nil {
			if isUniqueViolation(err) {
				return xerr.Wrap(err, "当前模型名称已存在")
			}
			return xerr.Wrapf(err, "创建模型配置失败")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return modelConfig, nil
}

func (r *ModelConfigRepository) Update(ctx context.Context, modelConfig *models.ModelConfig) (*models.ModelConfig, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if modelConfig.IsDefault {
			if err := tx.Model(&models.ModelConfig{}).
				Where("user_id = ? AND type = ?", modelConfig.UserID, modelConfig.Type).
				Update("is_default", false).Error; err != nil {
				return xerr.Wrapf(err, "更新默认模型配置失败")
			}
		}
		if err := tx.Save(&modelConfig).Error; err != nil {
			if isUniqueViolation(err) {
				return xerr.Wrap(err, "当前模型名称已存在")
			}
			return xerr.Wrapf(err, "更新模型配置失败")
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	
	return modelConfig, nil
}

func (r *ModelConfigRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", ID).Delete(&models.ModelConfig{})
	if result.Error != nil {
		return xerr.Wrap(result.Error, "删除模型配置失败")
	}
	if result.RowsAffected == 0 {
		return xerr.NotFound("模型配置不存在")
	}

	return nil
}

func (r *ModelConfigRepository) List(ctx context.Context, userID uuid.UUID, modelType *domain.ModelType) ([]*models.ModelConfig, error) {
	var rows []*models.ModelConfig

	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID)
	if modelType != nil {
		query = query.Where("type = ?", *modelType)
	}
	if err := query.Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, xerr.Wrapf(err, "查询模型配置失败")
	}
	return rows, nil
}

func (r *ModelConfigRepository) FindByID(ctx context.Context, userID uuid.UUID, configID uuid.UUID) (*models.ModelConfig, error) {
	config := &models.ModelConfig{}
	err := r.db.WithContext(ctx).Where("id = ? and user_id = ? ", configID, userID).First(config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NotFound("模型配置不存在")
	}
	if err != nil {
		return nil, xerr.Wrapf(err, "查询模型配置失败")
	}
	return config, nil
}
