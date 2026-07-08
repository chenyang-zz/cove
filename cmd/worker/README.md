# cmd/worker

Cove 后台任务处理器，基于 [asynq](https://github.com/hibiken/asynq) + Redis 消费异步任务。

## 运行

```bash
make worker
# 或直接
go run ./cmd/worker
```

需要与 `cmd/api` 配合使用，另开终端运行。

## 队列

| 队列 | 并发 | 用途 |
|---|---|---|
| `default` | 5 | 默认任务 |
| `parse` | 3 | 文档 / 图片解析 |
| `memory` | 3 | 记忆提取与合并 |
| `research` | 1 | 研究任务 |
| `beat` | 1 | 调度器写入的定时任务 |

## 已注册任务

- `parse:document` — 文档解析与分块
- `parse:image` — 图片内容提取
- `memory:extract` — 记忆提取
- `memory:consolidate` — 每日记忆合并（由 scheduler 触发）
- `research:run` — 研究任务执行

## 相关

- 任务注册：`internal/worker/tasks/`
- 任务处理器：`internal/worker/`
- asynq Redis 连接：`internal/infrastructure/queue/redis/`
