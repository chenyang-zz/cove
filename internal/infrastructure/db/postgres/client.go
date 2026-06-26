package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/boxify/api-go/internal/xerr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	URL               string
	MinConns          int32
	MaxConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type Client struct {
	pool *pgxpool.Pool
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	poolCfg, err := poolConfig(cfg)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, xerr.Wrapf(err, "创建 Postgres 连接池失败")
	}
	client := &Client{pool: pool}
	if err := client.Verify(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return client, nil
}

func (c *Client) Verify(ctx context.Context) error {
	if c == nil || c.pool == nil {
		return xerr.BadRequest("Postgres 客户端未初始化")
	}
	if err := c.pool.Ping(ctx); err != nil {
		return xerr.Wrapf(err, "验证 Postgres 连接失败")
	}
	return nil
}

func (c *Client) Close() {
	if c == nil || c.pool == nil {
		return
	}
	c.pool.Close()
}

func (c *Client) Pool() *pgxpool.Pool {
	if c == nil {
		return nil
	}
	return c.pool
}

func (c *Client) Tx(ctx context.Context, fn func(pgx.Tx) error) error {
	if fn == nil {
		return xerr.BadRequest("Postgres 事务函数不能为空")
	}
	if c == nil || c.pool == nil {
		return xerr.BadRequest("Postgres 客户端未初始化")
	}
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return xerr.Wrapf(err, "开启 Postgres 事务失败")
	}
	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return errors.Join(err, xerr.Wrapf(rollbackErr, "回滚 Postgres 事务失败"))
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return xerr.Wrapf(err, "提交 Postgres 事务失败")
	}
	return nil
}

func poolConfig(cfg Config) (*pgxpool.Config, error) {
	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		return nil, xerr.BadRequest("Postgres 连接 URL 不能为空")
	}
	poolCfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, xerr.Wrapf(err, "解析 Postgres 连接配置失败")
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}
	if cfg.HealthCheckPeriod > 0 {
		poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	}
	return poolCfg, nil
}
