# cmd/api

Cove HTTP API 服务入口，基于 Gin 框架，提供对话、RAG、Agent、记忆、MCP 等 RESTful 与 SSE 端点。

## 运行

```bash
make api
# 或直接
go run ./cmd/api
```

默认监听 `:8000`，可通过 `configs/config.yml` 中的 `http.port` 调整。

## 依赖

启动前请确保：

- PostgreSQL、Elasticsearch、Neo4j、Redis 已启动（见根目录 `deployments/docker-compose.yml`）
- 配置已复制：`cp configs/config.yml.example configs/config.yml`
- 迁移已执行：`make migration`

## 功能

- 加载配置与初始化 `ServiceContext`（依赖注入容器）
- 注册 Gin 路由、JWT 中间件、CORS
- 暴露 `/docs` Swagger UI（`docs.enabled: true` 时）
- 优雅关闭（需自行扩展信号处理，当前版本未内置）

## 相关

- 传输层实现：`internal/transport/http/`
- 服务上下文：`internal/svc/context.go`
