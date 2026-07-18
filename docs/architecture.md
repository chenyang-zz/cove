<p align="center">
  <img src="../logo/cove-logo/cove-logo-144x144.png" alt="Cove" width="144" height="144" />
</p>

# Cove Agent Platform

Cove 是一个 AI 助手平台的后端：对话、RAG、Agent、记忆、MCP——全部整合在一个 Go 代码库中。整体采用分层架构，HTTP 入口使用 Gin，但 Gin 被严格限制在 `transport/http` 层，不会泄漏到 domain、repository 或 infrastructure 包中。

## 技术栈

| 层 | 技术 |
|---|---|
| **语言** | Go 1.25 |
| **HTTP** | Gin（仅传输层，不侵入业务） |
| **数据库** | PostgreSQL (pgx + GORM) |
| **搜索** | Elasticsearch 8.x（向量 + BM25 混合） |
| **图** | Neo4j 5.x |
| **队列** | Redis + asynq |
| **LLM** | Anthropic / OpenAI 多 Provider |
| **认证** | JWT |
| **存储** | 腾讯云 COS（本地 fallback） |
| **可观测性** | slog + OpenTelemetry |
| **代码生成** | 内置注解扫描器（路由、仓库、OpenAPI） |

## 依赖方向

```
        ┌─────────────────────────────────────────────┐
        │              cmd/ (入口)                      │
        │   api │ worker │ scheduler │ migration │ codegen │
        └────────────────────┬────────────────────────┘
                             │ 初始化
                             ▼
        ┌─────────────────────────────────────────────┐
        │          internal/svc/ (DI 容器)              │
        │   ServiceContext 聚合仓库、配置、基础设施      │
        └────────────────────┬────────────────────────┘
                             │ 注入
         ┌───────────────────┼───────────────────┐
         ▼                   ▼                   ▼
┌────────────────┐ ┌─────────────────┐ ┌─────────────────────┐
│ transport/http │ │     logic/      │ │   internal/core/    │
│   HTTP 传输     │ │   业务编排       │ │    核心业务能力       │
│  Gin/DTO/中间件 │ │ 跨仓库 & domain  │ │ agent/llm/rag/...   │
└────────────────┘ └────────┬────────┘ └─────────────────────┘
                            │
                   ┌────────┴────────┐
                   ▼                 ▼
           ┌──────────────┐ ┌─────────────────┐
           │  repository/ │ │     domain/      │
           │  数据访问     │ │   领域类型/事件   │
           │ GORM/Neo4j/ES│ │   框架无关契约    │
           └──────┬───────┘ └─────────────────┘
                  │
                  ▼
           ┌──────────────────┐
           │  infrastructure/  │
           │  外部适配器        │
           │ db/es │ db/neo4j │
           │ llm   │ queue    │
           │ storage │ realtime│
           └──────────────────┘
```

## 模块职责

### transport/http
- Gin 路由、中间件链、请求/响应 DTO。
- 负责 JWT 认证、CORS、SSE 流式响应头设置。
- 路由注册由 `cmd/codegen` 自动生成，避免手写重复。
- **不引入**任何业务类型。

### logic
- 用例编排层：聚合多个 repository、调用 domain 服务、触发事件。
- 持有 `svc.ServiceContext`，通过构造函数注入。
- 不直接访问 infrastructure 细节。

### domain
- 纯领域类型与领域事件（发布订阅模式）。
- 零外部依赖（framework-free），可被任何上层引用。

### repository
- 持久化接口 + GORM / Neo4j / Elasticsearch 的具体实现。
- SQL 由 `db/queries/` 目录维护，通过代码生成产出类型安全访问。

### infrastructure
- 适配外部系统与中间件：PostgreSQL、Elasticsearch、Neo4j、Redis、COS、LLM Provider。
- 每个适配器独立子包，互不依赖，可单独 mock 测试。

## 核心能力（internal/core）

横切业务能力包，不随 HTTP/schema 频繁变化：

| 包 | 职责 |
|---|---|
| `tool` | 业务无关的工具描述、注册和调用能力 |
| `agent` | Agent 调度与工具调用（tool dispatch） |
| `llm` | LLM Provider 抽象层，支持 Anthropic / OpenAI 多后端 |
| `rag/` | 完整检索增强生成引擎（8 个子包） |
| `memory` | 长期记忆提取、合并与召回 |
| `mcp` | Model Context Protocol 集成 |
| `prompt` | 模板渲染（文件系统 / 内存 / 向后兼容 fallback） |
| `security` | JWT、加解密、密钥管理 |

### RAG 引擎

`internal/core/rag/` 包含 8 个子包：

```
rag/
├── chunker/        # tiktoken 分块（parent/child）
├── classifier/     # LLM 自动标签（非阻塞降级）
├── documentparse/  # 多格式文本提取
├── imagecompress/  # 模型输入预处理
├── imagedescribe/  # 视觉模型结构化描述
├── prompt/         # RAG 提示词模板（嵌入）
├── search/         # 向量 + BM25 混合召回 + 重排序
└── webcrawl/       # 网页抓取 + SSRF 防护
```

## 数据流

### RAG 入库流水线

```
Source → Crawl → Parse → Describe → Compress → Chunk → Embed
                                                          ↓
  Answer ← Classify ← Rerank ← Search ← Index ◄──────────┘
```

| 步骤 | 包 | 职责 |
|------|------|------|
| 1. Crawl | rag/webcrawl | 抓取 + 重定向跟踪 + SSRF 防护 |
| 2. Parse | rag/documentparse | TXT/MD/HTML/DOCX/PDF 文本提取 |
| 3. Describe | rag/imagedescribe | 视觉模型结构化描述 |
| 4. Compress | rag/imagecompress | 模型输入预处理 |
| 5. Chunk | rag/chunker | tiktoken parent/child 分块 |
| 6. Embed | llm/ (Provider) | 生成稠密向量 |
| 7. Index | Elasticsearch | Bulk upsert 写入 chunk 索引 |
| 8. Search | rag/search | 向量 + BM25 混合召回 |
| 9. Rerank | rag/search | 分数归一化 + 重排序 |
| 10. Classify | rag/classifier | LLM 自动标签（非阻塞） |
| 11. Answer | agent | 引用参考 + 生成回答 |

所有提示词模板位于 `internal/core/rag/prompt/`，通过 `internal/core/prompt/` 统一渲染。

### 对话流式路径

```
Client → Gin SSE → logic.Chat → agent.Run
                                   │
                    ┌──────────────┼──────────────┐
                    ▼              ▼              ▼
               rag.search    mcp.ToolCall    memory.Recall
                    │              │              │
                    └──────────────┼──────────────┘
                                   ▼
                             llm.Stream
                                   │
                                   ▼
                           SSE Event → Client
```

## 异步任务

通过 asynq + Redis 驱动后台任务（`cmd/worker`）：

| 任务 | 描述 |
|---|---|
| `parse:document` | 文档解析与分块 |
| `parse:image` | 图片内容提取 |
| `memory:extract` | 记忆提取 |
| `memory:consolidate` | 每日记忆合并 |
| `research:run` | 研究任务执行 |

Scheduler（`cmd/scheduler`）使用 Cron 定时触发任务，例如每日记忆合并。

## 配置与启动

### 配置加载

```
configs/config.yml → internal/config.Load → Config 结构体
                                          │
                          环境变量覆盖 (env://)
                          默认值兜底 (defaultConfig)
```

关键配置项：

```yaml
database:
  url: postgres://cove:cove@localhost:5432/cove
rag:
  chunk_index: cove_chunks
  embedding_dim: 1024
llm:
  provider: anthropic   # or openai
  api_key: "${LLM_API_KEY}"
jwt:
  secret: "${JWT_SECRET}"
```

### 启动接受路径

以下 `make` 命令从 Cove 工作区根目录运行；Docker 与直接 Go 命令从 `packages/server/` 运行。

1. `docker compose -f deployments/docker-compose.yml up -d` — 启动基础服务（PG / ES / Neo4j / Redis）。
2. `make migration` — 执行数据库迁移。
3. `make api` — 启动 HTTP 服务（`:8000`）。
4. 注册/登录 → 配置模型 → 上传内容 → 流式对话。

## 可观测性

- **日志**：Go 标准 `slog`，结构化输出。
- **链路追踪**：OpenTelemetry，覆盖 HTTP、LLM 调用、RAG 各阶段。
- **错误分级**：`internal/xerr` 提供带错误码、上下文信息的领域错误。

## 代码生成

`cmd/codegen/` 扫描 Go 注解自动生成样板代码：

| 命令 | 产物 |
|---|---|
| `make gen-route` | Gin 路由注册 |
| `make gen-repository` | 类型安全仓储 |
| `make gen-docs` | OpenAPI 3.0 规范 |

生成的产物统一置于 `internal/mapper/`、`docs/openapi.json`。

## 项目结构

```
.
├── cmd/                # 入口
│   ├── api/            # HTTP 服务
│   ├── worker/         # 后台任务
│   ├── scheduler/      # Cron 调度
│   ├── migration/      # 数据库迁移
│   └── codegen/        # 代码生成器
├── configs/            # 配置模板
├── deployments/        # Docker Compose
├── db/                 # 迁移脚本 & SQL 查询
├── docs/               # 架构文档 & OpenAPI
├── internal/
│   ├── config/         # 配置加载
│   ├── core/           # 核心业务能力
│   ├── domain/         # 领域类型
│   ├── infrastructure/ # 外部适配器
│   ├── logic/          # 业务编排
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
└── README.md
```
