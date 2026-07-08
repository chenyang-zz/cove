# 更新日志

所有对本项目的显著变更都会记录在此文件。

格式基于 [_keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增

- `codegen` 文档全面扩充：生成管线、模板清单、文件保护机制、子命令 Flag 与产物示例
- 新增 `CONTRIBUTING.md`、`CODE_OF_CONDUCT.md`、`SECURITY.md`
- 新增 `.github/` Issue 与 PR 模板、CI 工作流
- 新增 `LICENSE` 文件（MIT）

### 文档

- 修正 Repository 接口签名（FindByID / Update / UpdateFields）
- 修正 PostgreSQL 示例结构体名与参数名
- 移除 handler 示例中不存在的 DO NOT EDIT 头
- 澄清文件保护策略：repository 有 header、handler/logic 靠方法追加

---

## [0.1.0] - 2025-XX-XX

### 新增

- **对话** — 基于 SSE 的流式聊天，多轮上下文管理
- **RAG 引擎** — 完整的 11 步入库流水线：抓取、解析、描述、压缩、分块、嵌入、索引、检索、排序、分类、回答
- **Agent 编排** — ReAct 双路径（function calling / 文本 ReAct），prompt 模板化渲染，工具调用跨供应商归一化
- **记忆** — 长期记忆的提取、合并与召回
- **MCP 集成** — 通过 Model Context Protocol 连接外部工具
- **实时推送** — 基于 Redis 的事件流
- **文档处理** — 多格式解析：TXT、Markdown、HTML、DOCX、PDF
- **内容分类** — LLM 驱动的自动标签，支持优雅降级
- **API 文档** — 基于代码注解自动生成 OpenAPI 3.0 规范
