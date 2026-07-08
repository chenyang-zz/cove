# cmd/scheduler

Cove 定时任务调度器，基于 [asynq](https://github.com/hibiken/asynq) 的 `Scheduler`，将周期性任务写入 Redis 队列供 worker 消费。

## 运行

```bash
go run ./cmd/scheduler
```

通常作为常驻进程与 `cmd/api`、`cmd/worker` 一起部署。

## 已注册定时任务

| 任务 | 周期 | 说明 |
|---|---|---|
| `memory:consolidate` | `@daily` | 每日记忆合并，由 worker 执行 |

## 相关

- Worker 任务处理器：`internal/worker/`
- asynq Redis 连接：`internal/infrastructure/queue/redis/`
