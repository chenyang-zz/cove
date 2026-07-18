# Cove 多软件接入网关

Cove Gateway 是独立的数据面进程。API 负责账号、配对和绑定管理，Worker 负责执行 Agent Turn 与 Outbox 投递，Gateway 负责 Telegram 长轮询、飞书 WebSocket 和通用 Webhook 入站。

## 安全启用顺序

1. 部署包含网关模型的 API 与 Worker，并运行 `make migration`。
2. 通过 JWT 管理 API 创建渠道账号、批准私聊配对并配置群聊绑定。
3. 确认 API 和 Worker 健康后，将 `gateway.enabled` 设为 `true` 并启动 `make gateway`。
4. Docker Compose 中网关使用显式 profile：`docker compose -f deployments/docker-compose.yml --profile gateway up -d gateway`。

网关默认监听 `:8010`，健康检查为 `GET /healthz`。多实例共享 Redis 租约，每个 Telegram 或飞书账号同一时间只有一个 Receiver。

## 管理 API

所有 `/api/gateway` 路由都需要用户 JWT，并在仓储层按 Cove 用户隔离：

- `GET /api/gateway/providers`
- `GET|POST /api/gateway/accounts`
- `GET|PATCH|DELETE /api/gateway/accounts/{id}`
- `POST /api/gateway/accounts/{id}/test`
- `GET /api/gateway/accounts/{id}/pairings`
- `POST /api/gateway/pairings/{id}/approve|deny`
- `GET|POST /api/gateway/bindings`
- `GET|PATCH|DELETE /api/gateway/bindings/{id}`

账号凭据使用 Cove `SecretCipher` 加密，API 只返回掩码。Provider 描述包含凭据字段、可编辑设置和能力矩阵，可供后续管理界面自动生成表单。

## 通用 Webhook

创建 `provider=webhook` 的账号时，需要提供：

```json
{
  "provider": "webhook",
  "name": "个人助手",
  "credentials": {"signing_secret": "replace-with-a-random-secret"},
  "settings": {
    "callback_url": "https://example.com/cove/replies",
    "media_host_allowlist": ["media.example.com"]
  }
}
```

API 响应中的 `public_id` 用于入站地址：

```text
POST /gateway/v1/hooks/{public_id}
X-Cove-Timestamp: <Unix 秒>
X-Cove-Signature: HMAC_SHA256_HEX(secret, timestamp + "." + raw_body)
Content-Type: application/json
```

入站 body：

```json
{
  "event_id": "stable-event-id",
  "chat_type": "direct",
  "chat_id": "chat-42",
  "thread_id": "",
  "message_id": "message-9",
  "sender": {"id": "user-7", "display_name": "Cove User", "is_bot": false},
  "text": "你好",
  "mentioned": false,
  "media": [],
  "occurred_at": "2026-07-18T12:00:00Z"
}
```

签名时间默认只允许前后五分钟。合法请求在持久化后返回 HTTP `202`；重复的 `event_id` 返回原有 Cove `event_id`，且 `duplicate=true`，不会重复创建用户消息。

最终回复回调使用相同签名头，并额外携带 `X-Cove-Delivery-ID`：

```json
{
  "delivery_id": "stable-delivery-id",
  "route": {"account_id": "...", "chat_type": "direct", "chat_id": "chat-42"},
  "text": "最终回复",
  "reply_to": null
}
```

接收方必须以 `delivery_id` 去重。2xx 表示成功，429/5xx 表示明确可重试，其他 4xx 表示永久失败；发送已经开始但结果不确定时 Cove 不会盲目重发。

Webhook 媒体仅接受 HTTPS URL，主机必须在账号的 `media_host_allowlist` 中；DNS 解析、重定向和拨号阶段都会拒绝内网、回环、链路本地及其他非公开地址。

## 默认策略

- 未配对私聊只收到一小时有效的配对提示；每账号最多三个待处理请求。
- 群聊默认关闭。创建启用的群聊绑定后仍默认要求提及机器人。
- 私聊工具策略默认继承用户配置；群聊默认 `safe`，只开放当前时间和知识库检索。
- 绑定的 AgentConfig 优先于账号默认配置，账号默认配置优先于用户默认配置。
- 图片和文档先写入 Cove Storage；图片生成视觉描述，文档提取临时文本。解析失败会保留原文件并继续处理已有文本。
