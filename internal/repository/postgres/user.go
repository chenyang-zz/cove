package postgres

import (
	"context"
	"errors"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user repository.User) (repository.User, error) {
	if r == nil || r.db == nil {
		return repository.User{}, xerr.BadRequest("用户仓储未初始化")
	}
	model := userToModel(user)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		if isUniqueViolation(err) {
			return repository.User{}, xerr.UserExists()
		}
		return repository.User{}, xerr.Wrapf(err, "创建用户失败")
	}
	return userFromModel(model), nil
}

func (r *UserRepository) FindByLogin(ctx context.Context, login string) (repository.User, error) {
	if r == nil || r.db == nil {
		return repository.User{}, xerr.BadRequest("用户仓储未初始化")
	}
	var model models.User
	err := r.db.WithContext(ctx).
		Where("username = ? OR email = ?", login, login).
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.User{}, xerr.NotFound("用户不存在")
	}
	if err != nil {
		return repository.User{}, xerr.Wrapf(err, "查询用户失败")
	}
	return userFromModel(model), nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (repository.User, error) {
	if r == nil || r.db == nil {
		return repository.User{}, xerr.BadRequest("用户仓储未初始化")
	}
	var model models.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.User{}, xerr.NotFound("用户不存在")
	}
	if err != nil {
		return repository.User{}, xerr.Wrapf(err, "查询用户失败")
	}
	return userFromModel(model), nil
}

func userToModel(user repository.User) models.User {
	return models.User{
		ID:             user.ID,
		Username:       user.Username,
		Nickname:       user.Nickname,
		Email:          user.Email,
		Avatar:         user.Avatar,
		PasswordHash:   user.PasswordHash,
		BriefingSeenAt: nil,
	}
}

func userFromModel(model models.User) repository.User {
	return repository.User{
		ID:           model.ID,
		Username:     model.Username,
		Nickname:     model.Nickname,
		Email:        model.Email,
		Avatar:       model.Avatar,
		PasswordHash: model.PasswordHash,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
