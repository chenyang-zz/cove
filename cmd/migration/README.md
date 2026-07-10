# cmd/migration

Cove 数据库迁移工具，通过 GORM `AutoMigrate` 同步数据库表结构。

## 运行

```bash
make migration
# 或直接
go run ./cmd/migration
```

默认执行 `models.MigrationModels()` 注册的全部持久化模型迁移。

## 依赖

启动前请确保：

- PostgreSQL 已启动
- 配置已复制：`cp configs/config.yml.example configs/config.yml`
- `database.url` 中的连接串可正常连接

## 模型注册

需要迁移的 GORM 模型统一注册在 `internal/models/migration.go`。新增持久化模型时，
将模型指针加入 `models.MigrationModels()`，migration runner 本身无需修改。

## 相关

- 迁移实现：`internal/infrastructure/db/migration/`
- 模型注册：`internal/models/migration.go`
