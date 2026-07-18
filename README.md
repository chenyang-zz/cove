<p align="center">
  <img src="logo/cove-logo-full-bleed/cove-logo-full-bleed-192x192.png" alt="Cove" width="192" height="192" />
</p>

<h1 align="center">Cove API — Go</h1>

<p align="center">
  <b>Cove 是一个 AI 助手平台后端</b><br/>
  对话、RAG、Agent、记忆、MCP——全部整合在一个 Go 代码库中。
</p>

<p align="center">
  <!-- Badges -->
  <img src="https://img.shields.io/github/go-mod/go-version/chenyang-zz/cove-api?logo=go&logoColor=white&style=flat" alt="Go version" />
  <img src="https://img.shields.io/github/v/release/chenyang-zz/cove-api?style=flat&color=blue" alt="Release" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat" alt="License" />
</p>

<p align="center">
  <a href="#%E6%A0%87%E5%BF%97">特性</a> ·
  <a href="#%E6%8A%80%E6%9C%AF%E6%A0%88">技术栈</a> ·
  <a href="#%E5%BF%AB%E9%80%9F%E5%BC%80%E5%A7%8B">快速开始</a> ·
  <a href="#%E6%9E%B6%E6%9E%84">架构</a> ·
  <a href="#rag-%E7%AE%A1%E7%BA%BF">RAG 管线</a> ·
  <a href="#%E9%85%8D%E7%BD%AE">配置</a> ·
  <a href="#%E5%BC%80%E5%8F%91">开发</a> ·
  <a href="#%E6%B5%8B%E8%AF%95">测试</a> ·
  <a href="https://github.com/chenyang-zz/cove-api/blob/main/docs/architecture.md">文档</a>
</p>

---

## 特性

- **对话** — 基于 SSE 的流式聊天，多轮上下文管理
- **RAG 引擎** — 完整的检索增强生成：抓取、解析、分块、嵌入、检索、排序
- **Agent 编排** — ReAct 双路径（function calling / 文本 ReAct），prompt 模板化渲染，工具调用跨供应商归一化
- **记忆** — 长期记忆的提取、合并与召回
- **MCP 集成** — 通过 Model Context Protocol 连接外部工具
- **多软件网关** — Telegram、飞书与双向 HMAC Webhook，共用对话、权限与可靠投递能力
- **实时推送** — 基于 Redis 的事件流
- **文档处理** — 多格式解析：TXT、Markdown、HTML、DOCX、PDF
- **内容分类** — LLM 驱动的自动标签，支持优雅降级
- **API 文档** — 基于代码注解自动生成 OpenAPI 3.0 规范，内置 Swagger UI

## 技术栈

| 层 | 技术 |
|---|---|
| **语言** | Go 1.25 |
| **HTTP** | Gin（仅传输层，不侵入 domain） |
| **数据库** | PostgreSQL (pgx + GORM) |
| **搜索** | Elasticsearch 8.x（向量 + BM25 混合） |
| **图数据库** | Neo4j 5.x |
| **队列** | Redis + asynq |
| **LLM** | Anthropic / OpenAI |
| **认证** | JWT |
| **存储** | 腾讯云 COS（本地 fallback） |
| **可观测性** | slog + OpenTelemetry |

## 快速开始

### 前置条件

- Go 1.25+
- Docker & Docker Compose

### 1. 启动依赖服务

```bash
docker compose -f deployments/docker-compose.yml up -d
```

| 服务 | 端口 |
|---|---|
| PostgreSQL | 5432 |
| Elasticsearch | 9200 |
| Neo4j | 7474 (HTTP), 7687 (Bolt) |
| Redis | 6379 |

### 2. 配置

```bash
cp configs/config.yml.example configs/config.yml
# 编辑 configs/config.yml，填入 LLM 密钥和连接串
```

### 3. 数据库迁移

```bash
make migration
```

### 4. 运行

```bash
make api       # API 服务 :8000
make worker    # 后台 worker（另开终端）
make gateway   # 可选：消息网关 :8010，需先在配置中启用
```

## 架构

```
transport/http/    →  Gin 路由、中间件、请求/响应 DTO
    ↓
logic/             →  跨 repository 与 domain 的业务编排
    ↓
repository/        →  数据访问（GORM / Neo4j / Elasticsearch）
    ↓
domain/            →  领域类型、事件、接口
    ↓
infrastructure/    →  外部适配器（PostgreSQL / Elasticsearch / Neo4j / Redis / COS / LLM）
```

横切关注点（LLM、记忆、RAG、MCP、安全）位于 `internal/core/`，通过单一的 `ServiceContext` 注入 — 参见 `internal/svc/context.go`。

### 核心包

```
internal/core/
├── tool/           # 业务无关的工具描述、注册和调用能力
├── agent/          # Agent 编排与工具调度
│   ├── react/          # ReAct 编排（function calling / 文本推理 双路径）
│   └── prompt/         # Agent 提示词模板与变量结构
├── llm/            # LLM Provider 抽象（Client / ToolCallingClient / Message）
├── rag/            # 检索增强生成引擎
│   ├── chunker/        # tiktoken 感知的 parent/child 分块
│   ├── classifier/     # LLM 内容分类
│   ├── documentparse/  # 多格式文本提取
│   ├── imagecompress/  # 模型输入预处理
│   ├── imagedescribe/  # 视觉模型结构化描述
│   ├── prompt/         # RAG 提示词模板（嵌入产物）
│   ├── search/         # 向量 + BM25 混合检索
│   └── webcrawl/       # 网页抓取，含 SSRF 防护
├── memory/         # 长期记忆提取与合并
├── mcp/            # Model Context Protocol 集成
├── prompt/         # 模板渲染（文件系统、内存、向后兼容 fallback）
├── security/       # JWT、加解密、密钥管理
├── id/             # ID 生成器
├── jsonx/          # JSON 解析增强
└── valuex/         # 值类型工具
```

## RAG 管线

Cove 的 11 步入库流水线将原始来源转换为可检索的知识：

```
Source
  │
  ▼
1. Crawl       ──── webcrawl/     抓取，含重试、重定向跟踪、SSRF 防护
  │
  ▼
2. Parse       ──── documentparse/ 从 TXT/MD/HTML/DOCX/PDF 提取文本
  │
  ▼
3. Describe    ──── imagedescribe/ 视觉模型生成描述、OCR、物体、场景
  │
  ▼
4. Compress    ──── imagecompress/ 缩放与重编码，适配模型输入
  │
  ▼
5. Chunk       ──── chunker/       基于 tiktoken 的 parent/child 分块
  │
  ▼
6. Embed       ──── (provider)     通过 LLM Provider 生成稠密向量
  │
  ▼
7. Index       ──── Elasticsearch  Bulk upsert 写入 chunk 索引
  │
  ▼
8. Search      ──── search/        向量 + BM25 混合召回
  │
  ▼
9. Rerank      ──── search/        分数归一化与重排序
  │
  ▼
10. Classify   ──── classifier/    LLM 自动标签（非阻塞）
  │
  ▼
11. Answer     ──── agent/         引用参考，生成回答
```

所有提示词模板位于 `internal/core/rag/prompt/`，由 `internal/core/prompt/` 统一渲染。

> 详细流水线步骤与数据流图见 [`docs/architecture.md`](docs/architecture.md)。

## 配置

`configs/config.yml` 关键配置节：

```yaml
app:
  env: development

http:
  host: 0.0.0.0
  port: 8000

docs:
  enabled: true
  path: /docs
  spec_path: /docs/openapi.json
  title: Cove API
  version: 0.1.0

database:
  url: postgres://cove:cove@localhost:5432/cove?sslmode=disable

redis:
  addr: localhost:6379

elasticsearch:
  url: http://localhost:9200

neo4j:
  uri: bolt://localhost:7687
  username: "neo4j"
  password: "change-me"

secret_key: "0123456789abcdef0123456789abcdef"

jwt:
  secret: change-me
  access_token_ttl: 168h

storage:
  backend: local          # 或 cos
  dir: ./storage
  cos:
    bucket_url: ""
    secret_id: ""
    secret_key: ""
    base_url: ""

llm:
  provider: openai        # 或 anthropic
  model: gpt-4o-mini
  embedding_model: text-embedding-3-small
  base_url: https://api.openai.com/v1
  api_key: ""

rag:
  embedding_dim: 1024
  embedding_batch_size: 10

memory:
  name_sim_gate: 0.8
  llm_merge_confidence: 0.8
  community_clustering_max_iterations: 10
  community_vote_sem_weight: 0.6
  community_vote_rel_weight: 0.4
  community_merge_threshold: 0.85

agent:
  max_personas: 200

gateway:
  enabled: false             # 完成迁移并启动 API/Worker 后再开启
  host: 0.0.0.0
  port: 8010
  reconcile_interval: 30s
  lease_ttl: 45s
  webhook_signing_window: 5m
  callback_timeout: 10s
  max_request_bytes: 1048576
  max_media_bytes: 20971520
```

完整配置项与默认值见 [`configs/config.yml.example`](configs/config.yml.example)。

## 开发

### 代码生成

Cove 内置代码生成器（`cmd/codegen/`），扫描 Go 注解自动生成：

| 命令 | 产物 |
|---|---|
| `make gen-route` | Gin 路由注册 |
| `make gen-repository MODEL=User LABEL=用户` | 类型安全仓储 |
| `make gen-docs` | OpenAPI 3.0 规范 |

### API 路由

所有路由挂载在 `/api/` 下：

| 路径 | 说明 |
|---|---|
| `/api/health` | 健康检查（公开） |
| `/api/auth` | 注册 / 登录 |
| `/api/models` | 模型配置 |
| `/api/chat` | 流式对话 |
| `/api/conversations` | 会话管理 |
| `/api/documents` | 文档 CRUD |
| `/api/knowledge-bases` | 知识库管理 |
| `/api/agents` | Agent 配置 |
| `/api/mcp-servers` | MCP 服务集成 |
| `/api/gateway` | 渠道账号、配对与路由绑定管理 |

已认证路由受 JWT 中间件保护。

### 异步任务

基于 asynq + Redis 驱动：

| 任务 | 说明 |
|---|---|
| `parse:document` | 文档解析与分块 |
| `parse:image` | 图片内容提取 |
| `memory:extract` | 记忆提取 |
| `memory:consolidate` | 每日记忆合并 |
| `research:run` | 研究任务执行 |
| `gateway:turn` | 串行执行外部渠道对话 |
| `gateway:deliver` | 可靠投递渠道最终回复 |

## 测试

项目使用 Go 标准测试框架，核心包测试基于本地 fake 实现，无需外部依赖即可运行：

```bash
go test ./...              # 全量测试
go test ./internal/core/... # 核心业务能力
go test ./internal/agent/... # Agent 编排
```

每个测试函数上方均附有中文注释说明验证点。

## 项目结构

```
.
├── cmd/                # 入口
│   ├── api/            # HTTP 服务
│   ├── worker/         # 后台处理器
│   ├── gateway/        # 独立消息网关
│   ├── scheduler/      # Cron 调度器
│   ├── migration/      # 数据库迁移
│   └── codegen/        # 代码生成工具
├── configs/            # 配置模板
├── deployments/        # Docker Compose
├── db/                 # 迁移脚本 & 查询
├── docs/               # 架构文档 & OpenAPI
├── internal/
│   ├── config/         # 配置加载
│   ├── core/           # 核心业务能力
│   ├── domain/         # 领域类型 & 事件
│   ├── infrastructure/ # 外部适配器
│   │   ├── db/             # PostgreSQL 连接
│   │   ├── id/             # 分布式 ID
│   │   ├── jsonrepair/     # JSON 修复
│   │   ├── llm/            # LLM Provider 实现
│   │   ├── queue/          # Redis 队列
│   │   ├── realtime/       # 实时推送
│   │   ├── security/       # 安全适配器
│   │   └── storage/        # 对象存储
│   ├── logic/          # 业务逻辑层
│   ├── mapper/         # 对象映射（生成）
│   ├── models/         # GORM 模型
│   ├── observability/  # 日志 & 追踪
│   ├── prompts/        # 提示词定义
│   ├── repository/     # 数据访问
│   ├── svc/            # ServiceContext (DI)
│   ├── transport/http/ # HTTP 传输层
│   ├── util/           # 工具函数
│   ├── worker/         # 任务处理器
│   └── xerr/           # 错误定义
├── Makefile
└── README.md
```

## 使用示例

### 注册与登录

```bash
curl -X POST http://localhost:8000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"your-password"}'
```

```bash
curl -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"your-password"}'
# → {"code":0,"data":{"token":"eyJ..."}}
```

### 流式对话

登录后使用返回的 JWT 调用对话接口：

```bash
curl -N http://localhost:8000/api/chat/stream \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{"conversation_id":"<uuid>","message":"介绍一下你自己"}'
# → SSE 流: data: {"content":"...\n","finish":false}
```

### 文档上传

```bash
curl -X POST http://localhost:8000/api/documents \
  -H "Authorization: Bearer <your-jwt-token>" \
  -F "file=@/path/to/your.pdf" \
  -F "knowledge_base_id=<uuid>"
# → {"code":0,"data":{"id":"<uuid>","status":"processing"}}
```

> 完整 API 文档与所有路由的 request/response schema 见自动生成的 [OpenAPI 规范](docs/openapi.json)（运行时通过 `/docs` 查看 Swagger UI）。

## 文档

| 文档 | 说明 |
|---|---|
| [架构文档](docs/architecture.md) | 技术栈、分层架构、RAG 数据流、启动路径 |
| [OpenAPI 规范](docs/openapi.json) | 自动生成的 API 参考（OpenAPI 3.0.3） |
| [多软件网关](docs/gateway.md) | Provider、配对、Webhook 协议与启用流程 |
| [代码生成器](cmd/codegen/README.md) | codegen 工具完整使用指南 |
| [贡献指南](CONTRIBUTING.md) | 分支命名、提交规范、PR 清单 |
| [更新日志](CHANGELOG.md) | 版本变更记录 |
| [安全政策](SECURITY.md) | 漏洞报告流程 |
| [行为准则](CODE_OF_CONDUCT.md) | 社区行为准则 |
| [许可证](LICENSE) | MIT |

## 贡献

欢迎提交 Issue 和 Pull Request！请先阅读 [贡献指南](CONTRIBUTING.md)，了解分支命名、提交规范与 PR 清单。

## 许可证

[MIT](LICENSE) © Cove Team

---

<p align="center">
  <img src="logo/cove-logo/cove-logo-64x64.png" alt="Cove" width="64" height="64" />
</p>

<p align="center">
  Built with Go · LLM-powered · 欢迎贡献
</p>
