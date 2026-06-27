package redis

import (
	"context"
	"strings"

	goredis "github.com/redis/go-redis/v9"

	"github.com/boxify/api-go/internal/xerr"
)

type Config struct {
	Addr     string
	Username string
	Password string
	DB       int
}

type Client struct {
	raw *goredis.Client
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	_ = ctx
	if strings.TrimSpace(cfg.Addr) == "" {
		return nil, xerr.BadRequest("Redis addr 配置无效")
	}
	return &Client{raw: goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	})}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if err := c.raw.Ping(ctx).Err(); err != nil {
		return xerr.Wrapf(err, "连接 Redis 失败")
	}
	return nil
}

func (c *Client) Close() error {
	if c == nil || c.raw == nil {
		return nil
	}
	if err := c.raw.Close(); err != nil {
		return xerr.Wrapf(err, "关闭 Redis 连接失败")
	}
	return nil
}

func (c *Client) Raw() *goredis.Client {
	return c.raw
}
