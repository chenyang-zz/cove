package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AgentConfigRepository struct {
	db *gorm.DB
}

func NewAgentConfigRepository(db *gorm.DB) repository.AgentConfigRepository {
	return &AgentConfigRepository{db: db}
}

func (r *AgentConfigRepository) Create(ctx context.Context, userID uuid.UUID, agentConfig *models.AgentConfig) (*models.AgentConfig, error) {
	if agentConfig == nil {
		return nil, xerr.BadRequest("智能体配置不能为空")
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockAgentConfigUser(tx, userID); err != nil {
			return err
		}
		agentConfig.Name = strings.TrimSpace(agentConfig.Name)
		if agentConfig.Name == "" {
			name, err := nextAgentConfigName(tx, userID)
			if err != nil {
				return err
			}
			agentConfig.Name = name
		}
		var defaults int64
		if err := tx.Model(&models.AgentConfig{}).
			Where("user_id = ? AND is_default = ?", userID, true).
			Count(&defaults).Error; err != nil {
			return xerr.Wrapf(err, "查询默认智能体配置失败")
		}
		agentConfig.UserID = userID
		agentConfig.IsDefault = defaults == 0
		if err := tx.Create(agentConfig).Error; err != nil {
			if isUniqueViolation(err) {
				return xerr.Conflict("智能体配置名称已存在")
			}
			return xerr.Wrapf(err, "创建智能体配置失败")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return agentConfig, nil
}

func (r *AgentConfigRepository) List(ctx context.Context, userID uuid.UUID) ([]*models.AgentConfig, error) {
	var rows []*models.AgentConfig

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, xerr.Wrapf(err, "查询智能体配置列表失败")
	}

	return rows, nil
}

// FindDefault 返回用户的默认配置；旧数据没有默认标记时会提升最近更新的一条配置。
func (r *AgentConfigRepository) FindDefault(ctx context.Context, userID uuid.UUID) (*models.AgentConfig, error) {
	var result *models.AgentConfig
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockAgentConfigUser(tx, userID); err != nil {
			return err
		}
		config, err := findDefaultAgentConfig(tx, userID)
		if err == nil {
			result = config
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return xerr.Wrapf(err, "查询默认智能体配置失败")
		}

		// 旧记录没有默认标记时，稳定地选择最近更新的一条并写回默认状态。
		config = &models.AgentConfig{}
		if err := tx.Where("user_id = ?", userID).
			Order("updated_at DESC, id DESC").
			First(config).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return xerr.NotFound("默认智能体配置不存在")
			}
			return xerr.Wrapf(err, "查询智能体配置失败")
		}
		if err := tx.Model(&models.AgentConfig{}).
			Where("id = ? AND user_id = ?", config.ID, userID).
			Update("is_default", true).Error; err != nil {
			return xerr.Wrapf(err, "修复默认智能体配置失败")
		}
		config.IsDefault = true
		result = config
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AgentConfigRepository) FindByID(ctx context.Context, userID uuid.UUID, agentConfigID uuid.UUID) (*models.AgentConfig, error) {
	agentConfig := &models.AgentConfig{}
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", agentConfigID, userID).
		First(agentConfig).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NotFound("智能体配置不存在")
	}
	if err != nil {
		return nil, xerr.Wrapf(err, "查询智能体配置失败")
	}
	return agentConfig, nil
}

// SetDefault 原子地将指定配置设为当前用户的唯一默认配置。
func (r *AgentConfigRepository) SetDefault(ctx context.Context, userID uuid.UUID, agentConfigID uuid.UUID) (*models.AgentConfig, error) {
	var result *models.AgentConfig
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockAgentConfigUser(tx, userID); err != nil {
			return err
		}
		config := &models.AgentConfig{}
		if err := tx.Where("id = ? AND user_id = ?", agentConfigID, userID).First(config).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return xerr.NotFound("智能体配置不存在")
			}
			return xerr.Wrapf(err, "查询智能体配置失败")
		}
		if err := tx.Model(&models.AgentConfig{}).
			Where("user_id = ? AND is_default = ?", userID, true).
			Update("is_default", false).Error; err != nil {
			return xerr.Wrapf(err, "清除默认智能体配置失败")
		}
		if err := tx.Model(&models.AgentConfig{}).
			Where("id = ? AND user_id = ?", agentConfigID, userID).
			Update("is_default", true).Error; err != nil {
			return xerr.Wrapf(err, "设置默认智能体配置失败")
		}
		config.IsDefault = true
		result = config
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AgentConfigRepository) Update(ctx context.Context, userID uuid.UUID, agentConfig *models.AgentConfig) (*models.AgentConfig, error) {
	agentConfig.Name = strings.TrimSpace(agentConfig.Name)
	result := r.db.WithContext(ctx).
		Model(&models.AgentConfig{}).
		Where("id = ? AND user_id = ?", agentConfig.ID, userID).
		Omit("id", "user_id", "user", "is_default", "created_at", "updated_at").
		Updates(agentConfig)
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return nil, xerr.Conflict("智能体配置名称已存在")
		}
		return nil, xerr.Wrapf(result.Error, "更新智能体配置失败")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NotFound("智能体配置不存在")
	}
	return r.FindByID(ctx, userID, agentConfig.ID)
}

func (r *AgentConfigRepository) UpdateFields(ctx context.Context, userID uuid.UUID, agentConfigID uuid.UUID, agentConfig *models.AgentConfig, fields *repository.AgentConfigUpdateFields) (*models.AgentConfig, error) {
	columns := fields.Columns()
	if len(columns) == 0 {
		return nil, xerr.BadRequest("更新字段不能为空")
	}
	if containsAgentConfigColumn(columns, "name") {
		agentConfig.Name = strings.TrimSpace(agentConfig.Name)
		if agentConfig.Name == "" {
			return nil, xerr.BadRequest("智能体配置名称不能为空")
		}
	}
	result := r.db.WithContext(ctx).
		Model(&models.AgentConfig{}).
		Where("id = ? AND user_id = ?", agentConfigID, userID).
		Select(columns).
		Updates(agentConfig)
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return nil, xerr.Conflict("智能体配置名称已存在")
		}
		return nil, xerr.Wrapf(result.Error, "更新智能体配置失败")
	}
	if result.RowsAffected == 0 {
		return nil, xerr.NotFound("智能体配置不存在")
	}
	return r.FindByID(ctx, userID, agentConfigID)
}

func (r *AgentConfigRepository) Delete(ctx context.Context, userID uuid.UUID, agentConfigID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockAgentConfigUser(tx, userID); err != nil {
			return err
		}
		config := &models.AgentConfig{}
		if err := tx.Where("id = ? AND user_id = ?", agentConfigID, userID).First(config).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return xerr.NotFound("智能体配置不存在")
			}
			return xerr.Wrapf(err, "查询智能体配置失败")
		}
		if err := tx.Delete(&models.AgentConfig{}, "id = ? AND user_id = ?", agentConfigID, userID).Error; err != nil {
			return xerr.Wrapf(err, "删除智能体配置失败")
		}
		if !config.IsDefault {
			return nil
		}

		// 删除默认项后提升最近更新的剩余配置；没有剩余项时保持空集合。
		next := &models.AgentConfig{}
		if err := tx.Where("user_id = ?", userID).
			Order("updated_at DESC, id DESC").
			First(next).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return xerr.Wrapf(err, "查询后继默认智能体配置失败")
		}
		if err := tx.Model(&models.AgentConfig{}).
			Where("id = ? AND user_id = ?", next.ID, userID).
			Update("is_default", true).Error; err != nil {
			return xerr.Wrapf(err, "提升默认智能体配置失败")
		}
		return nil
	})
}

func findDefaultAgentConfig(tx *gorm.DB, userID uuid.UUID) (*models.AgentConfig, error) {
	config := &models.AgentConfig{}
	err := tx.Where("user_id = ? AND is_default = ?", userID, true).First(config).Error
	return config, err
}

func nextAgentConfigName(tx *gorm.DB, userID uuid.UUID) (string, error) {
	var names []string
	if err := tx.Model(&models.AgentConfig{}).
		Where("user_id = ?", userID).
		Pluck("name", &names).Error; err != nil {
		return "", xerr.Wrapf(err, "查询智能体配置名称失败")
	}
	if len(names) == 0 {
		return "默认配置", nil
	}

	existing := make(map[string]struct{}, len(names))
	for _, name := range names {
		existing[name] = struct{}{}
	}
	for sequence := len(names) + 1; ; sequence++ {
		candidate := fmt.Sprintf("智能体配置 %d", sequence)
		if _, ok := existing[candidate]; !ok {
			return candidate, nil
		}
	}
}

func containsAgentConfigColumn(columns []string, target string) bool {
	for _, column := range columns {
		if column == target {
			return true
		}
	}
	return false
}

func lockAgentConfigUser(tx *gorm.DB, userID uuid.UUID) error {
	user := &models.User{}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id = ?", userID).
		First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return xerr.NotFound("用户不存在")
		}
		return xerr.Wrapf(err, "锁定用户默认智能体配置失败")
	}
	return nil
}
