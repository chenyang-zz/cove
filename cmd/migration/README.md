# cmd/migration

Cove 数据库迁移工具，执行 `db/migrations/` 目录下的 SQL 迁移脚本。

## 运行

```bash
make migration
# 或直接
go run ./cmd/migration
```

默认执行所有待处理的向上迁移（`Up`）。

## 依赖

启动前请确保：

- PostgreSQL 已启动
- 配置已复制：`cp configs/config.yml.example configs/config.yml`
- `database.url` 中的连接串可正常连接

## 迁移文件

SQL 迁移脚本位于 `db/migrations/`，按版本号顺序执行。

## 相关

- 迁移实现：`internal/infrastructure/db/migration/`
- 迁移脚本：`db/migrations/`
