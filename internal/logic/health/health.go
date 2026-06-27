package health

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/infrastructure/db/postgres"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"golang.org/x/sync/errgroup"
)

type HealthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

func NewHealthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HealthLogic {
	return &HealthLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.health.health"),
	}
}

func (l *HealthLogic) Health() (*response.HealthResponse, error) {
	g, ctx := errgroup.WithContext(l.ctx)

	res := make([]*response.HealthItem, 0, 5)

	type checkItemType struct {
		serverName string
		run        func(context.Context) error
	}

	checkItems := []*checkItemType{
		{
			serverName: "redis",
			run: func(ctx context.Context) error {
				if l.svcCtx.Redis == nil {
					return xerr.BadRequest("Redis 客户端未初始化")
				}
				return l.svcCtx.Redis.Ping(ctx)
			},
		},
		{
			serverName: "neo4j",
			run: func(ctx context.Context) error {
				if l.svcCtx.Neo4j == nil {
					return xerr.BadRequest("Neo4j 客户端未初始化")
				}
				return l.svcCtx.Neo4j.Ping(ctx)
			},
		},
		{
			serverName: "elasticsearch",
			run: func(ctx context.Context) error {
				if l.svcCtx.Elasticsearch == nil {
					return xerr.BadRequest("Elasticsearch 客户端未初始化")
				}
				return l.svcCtx.Elasticsearch.Ping(ctx)
			},
		},
		{
			serverName: "postgres",
			run: func(c context.Context) error {
				return postgres.PingGormDB(c, l.svcCtx.GormDB)
			},
		},
		{
			serverName: "cos",
			run: func(ctx context.Context) error {
				if l.svcCtx.Storage == nil {
					return xerr.BadRequest("对象存储未初始化")
				}
				return l.svcCtx.Storage.Ping(ctx)
			},
		},
	}

	res = make([]*response.HealthItem, len(checkItems))
	for i, check := range checkItems {
		i, check := i, check
		g.Go(func() error {
			err := check.run(ctx)

			res[i] = &response.HealthItem{
				ServerName: check.serverName,
				IsHealthy:  err == nil,
			}

			if err != nil {
				res[i].Error = err.Error()
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, xerr.Wrap(err, "检查服务健康状态失败")
	}

	return &response.HealthResponse{List: res}, nil
}
