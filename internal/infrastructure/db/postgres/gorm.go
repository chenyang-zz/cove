package postgres

import (
	"context"
	"strings"

	"github.com/boxify/api-go/internal/xerr"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewGormDB(ctx context.Context, cfg Config) (*gorm.DB, error) {
	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		return nil, xerr.BadRequest("Postgres 连接 URL 不能为空")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		return nil, xerr.Wrapf(err, "打开 GORM Postgres 连接失败")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, xerr.Wrapf(err, "获取 GORM 底层数据库连接失败")
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, xerr.Wrapf(err, "验证 GORM Postgres 连接失败")
	}
	return db, nil
}

func PingGormDB(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return xerr.BadRequest("GORM Postgres 连接未初始化")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return xerr.Wrapf(err, "获取 GORM 底层数据库连接失败")
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return xerr.Wrapf(err, "验证 GORM Postgres 连接失败")
	}
	return nil
}
