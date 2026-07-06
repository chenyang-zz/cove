package llm

import (
	"context"
)

// Client 表示业务无关的模型客户端。
//
// 实现应同时提供兼容旧调用的纯文本 Invoke，以及携带工具调用、token 用量和停止原因的
// InvokeResult。Stream 仍只返回文本增量；Embed 和 EmbedOne 负责文本向量化。
type Client interface {
	// Invoke 执行一次非流式文本生成，并返回模型文本内容。
	Invoke(ctx context.Context, messages []*Message, opts ...ModelCallOption) (string, error)
	// InvokeResult 执行一次非流式生成，并返回结构化模型结果。
	InvokeResult(ctx context.Context, messages []*Message, opts ...ModelCallOption) (*LLMResult, error)
	// Stream 执行一次流式文本生成，并返回文本增量通道。
	Stream(ctx context.Context, messages []*Message, opts ...ModelCallOption) (<-chan string, error)
	// Embed 批量生成文本向量。
	Embed(ctx context.Context, texts []string, dimensions int, opts ...EmbeddingOption) ([][]float64, error)
	// EmbedOne 生成单条文本向量。
	EmbedOne(ctx context.Context, text string, dimensions int) ([]float64, error)
	// Vision(ctxt context.Context, prompt string) (string, error)
	// Rerank(ctx context.Context, query string, documents []string, top_n int) error
}
