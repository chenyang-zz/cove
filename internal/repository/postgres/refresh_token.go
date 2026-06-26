package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) repository.RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token repository.RefreshToken) (repository.RefreshToken, error) {
	if r == nil || r.db == nil {
		return repository.RefreshToken{}, xerr.BadRequest("刷新令牌仓储未初始化")
	}
	model := refreshTokenToModel(token)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		if isUniqueViolation(err) {
			return repository.RefreshToken{}, xerr.InvalidToken()
		}
		return repository.RefreshToken{}, xerr.Wrapf(err, "创建刷新令牌失败")
	}
	return refreshTokenFromModel(model), nil
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (repository.RefreshToken, error) {
	if r == nil || r.db == nil {
		return repository.RefreshToken{}, xerr.BadRequest("刷新令牌仓储未初始化")
	}
	var model models.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.RefreshToken{}, xerr.InvalidToken()
	}
	if err != nil {
		return repository.RefreshToken{}, xerr.Wrapf(err, "查询刷新令牌失败")
	}
	return refreshTokenFromModel(model), nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	if r == nil || r.db == nil {
		return xerr.BadRequest("刷新令牌仓储未初始化")
	}
	result := r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", revokedAt)
	if result.Error != nil {
		return xerr.Wrapf(result.Error, "撤销刷新令牌失败")
	}
	if result.RowsAffected == 0 {
		return xerr.InvalidToken()
	}
	return nil
}

func refreshTokenToModel(token repository.RefreshToken) models.RefreshToken {
	return models.RefreshToken{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		RevokedAt: token.RevokedAt,
		CreatedAt: token.CreatedAt,
		UpdatedAt: token.UpdatedAt,
	}
}

func refreshTokenFromModel(model models.RefreshToken) repository.RefreshToken {
	return repository.RefreshToken{
		ID:        model.ID,
		UserID:    model.UserID,
		TokenHash: model.TokenHash,
		ExpiresAt: model.ExpiresAt,
		RevokedAt: model.RevokedAt,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}
