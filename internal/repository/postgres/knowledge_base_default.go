package postgres

import (
	"context"
	"errors"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SetDefault 原子地将指定知识库设为当前用户的唯一默认知识库。
func (r *KnowledgeBaseRepository) SetDefault(ctx context.Context, userID uuid.UUID, knowledgeBaseID uuid.UUID) (*models.KnowledgeBase, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 锁定用户行，将同一用户的并发默认项切换串行化。
		user := &models.User{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").
			Where("id = ?", userID).
			First(user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return xerr.NotFound("用户不存在")
			}
			return xerr.Wrapf(err, "锁定用户默认知识库失败")
		}

		target := &models.KnowledgeBase{}
		if err := tx.Where("id = ? AND user_id = ?", knowledgeBaseID, userID).First(target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return xerr.NotFound("知识库不存在")
			}
			return xerr.Wrapf(err, "查询知识库失败")
		}

		// 先清除其他默认项，再设置目标项，保证事务提交后只有一个默认知识库。
		if err := tx.Model(&models.KnowledgeBase{}).
			Where("user_id = ? AND id <> ? AND is_default = ?", userID, knowledgeBaseID, true).
			Update("is_default", false).Error; err != nil {
			return xerr.Wrapf(err, "清除默认知识库失败")
		}
		if err := tx.Model(&models.KnowledgeBase{}).
			Where("id = ? AND user_id = ?", knowledgeBaseID, userID).
			Update("is_default", true).Error; err != nil {
			return xerr.Wrapf(err, "设置默认知识库失败")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, userID, knowledgeBaseID)
}
