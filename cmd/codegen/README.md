# codegen

`cmd/codegen` 是项目内置代码生成工具，用于根据 route 注释生成 handler/logic，根据 GORM model 生成 repository。

## 常用命令

```bash
go run ./cmd/codegen route
go run ./cmd/codegen route --dry-run
go run ./cmd/codegen route --check
go run ./cmd/codegen route --list

go run ./cmd/codegen repository -model Conversation -label 会话
go run ./cmd/codegen repository -model Message -label 消息 -scope conversation_id:conversations.id:user_id
go run ./cmd/codegen repository --list-models

go run ./cmd/codegen doctor
go run ./cmd/codegen doctor --format json
```

Makefile 同步提供快捷入口：

```bash
make gen-route
make gen-route DRY_RUN=1
make gen-route CHECK=1 FORMAT=json
make gen-repository MODEL=Conversation LABEL=会话
make gen-repository MODEL=Message LABEL=消息 SCOPE=conversation_id:conversations.id:user_id
```

## Route 注释协议

推荐使用 `@` 指令：

```go
// @auth(user_id)
// @description 重命名会话
// 修改当前用户拥有的会话标题。
// @summary 重命名会话
// @input request.RenameConversationRequest
// @response ConversationResponse
conversationRoutes.PATCH("/:conversation_id", conversation.RenameConversation)
```

SSE 路由必须声明事件类型：

```go
// @auth(user_id)
// @sse
// @event domain.AgentEvent
// @input request.ChatStreamRequest
chatRoutes.POST("/stream", chat.ChatStream)
```

旧协议 `routegen: ...` 继续兼容。

## Repository Scope

拥有 `UserID uuid.UUID` 的 model 使用直接作用域：

```bash
go run ./cmd/codegen repository -model Conversation -label 会话
```

没有 `UserID` 的 model 需要显式声明父级作用域：

```bash
go run ./cmd/codegen repository -model Message -label 消息 -scope conversation_id:conversations.id:user_id
```

`--list-models` 会标记哪些 model 可以直接生成，哪些需要补 `--scope`。

## 工具模式

- `--dry-run`：只预览 would-add / would-modify，不写入磁盘。
- `--check`：只检查生成结果是否过期；有待变更时返回非零退出码。
- `--verbose`：输出扫描诊断信息。
- `--format json`：输出结构化 JSON，适合 CI 或脚本消费。
- `doctor`：只扫描并报告协议、DTO、repository scope 等问题，不生成代码。
