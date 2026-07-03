package search

import (
	"context"
)

// ESClient 定义检索器需要的最小 Elasticsearch 查询能力。
type ESClient interface {
	Search(ctx context.Context, index string, query any) (map[string]any, error)
}

// Embedder 定义单文本向量化能力。
type Embedder interface {
	EmbedOne(ctx context.Context, text string, dimensions int) ([]float64, error)
}

// Reranker 定义候选文档重排能力。
//
// documents 的顺序来自第一阶段融合排序，RerankResult.Index 必须指向 documents 中的下标。
// 实现可以只返回前 topN 个结果；越界或重复下标会被 Search 忽略。
type Reranker interface {
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
}

// FilterBuilder 根据请求构造 ES filter。
//
// 返回的 filter 会同时用于向量召回和 BM25 召回；业务过滤规则由调用方自行实现。
type FilterBuilder func(ctx context.Context, req Input) ([]any, error)

// SourceDecoder 把 ES _source 解码成调用方需要的类型。
//
// decoder 返回错误时 Search 会终止并返回该错误，避免静默丢失业务元数据。
type SourceDecoder[T any] func(src map[string]any) (T, error)

// RerankDocumentBuilder 根据 ES _source 构造送入 reranker 的文档文本。
//
// 默认实现读取 content 字段。调用方可以覆盖为标题、摘要和结构化元数据拼接后的文本；
// builder 返回空字符串时仍会把该候选传给 reranker，由具体 reranker 适配器决定如何处理。
type RerankDocumentBuilder func(src map[string]any) string

// RerankResult 表示重排模型返回的候选下标和分数。
type RerankResult struct {
	// Index 是被重排 documents 中的候选下标。
	Index int
	// Score 是 reranker 返回的相关性分数，分数尺度由具体实现决定。
	Score float64
}

// Input 描述一次 RAG 检索请求的内部配置。
//
// RecallSize 控制两路召回池大小，最终仍由 TopK 裁剪。
// Filters 会透传给向量召回和 BM25 召回，业务过滤规则由调用方提供。
// MinVectorScore 启用后按 ES cosine 原始相关度门控，适合精确搜索场景。
// 其余未导出字段由 InputOption 写入，用于覆盖默认 embedder 和 rerank 配置。
type Input struct {
	Query          string
	TopK           int
	RecallSize     int
	Filters        []any
	MinVectorScore *float64
	embedder       Embedder
	reranker       Reranker
	rerankEnabled  *bool
	rerankWindow   int
	rerankTopK     int
	rerankFailOpen *bool
	rerankMinScore *float64
	rerankBuilder  RerankDocumentBuilder
}

// InputOption 修改单次 Search 请求配置。
type InputOption func(*Input)

// WithTopK 设置最终返回结果数量。
func WithTopK(topK int) InputOption {
	return func(req *Input) {
		if topK > 0 {
			req.TopK = topK
		}
	}
}

// WithInputRecallSize 设置单次请求的两路召回池大小。
func WithInputRecallSize(recallSize int) InputOption {
	return func(req *Input) {
		if recallSize > 0 {
			req.RecallSize = recallSize
		}
	}
}

// WithFilters 设置单次请求要透传给 ES 的 filter。
func WithFilters(filters []any) InputOption {
	return func(req *Input) {
		req.Filters = filters
	}
}

// WithMinVectorScore 设置最低 cosine 向量相关度门槛。
func WithMinVectorScore(minVectorScore float64) InputOption {
	return func(req *Input) {
		req.MinVectorScore = &minVectorScore
	}
}

// WithInputEmbedder 设置单次请求使用的向量化客户端。
func WithInputEmbedder(embedder Embedder) InputOption {
	return func(req *Input) {
		if embedder != nil {
			req.embedder = embedder
		}
	}
}

// WithInputReranker 设置单次请求使用的重排器。
//
// reranker 为 nil 时忽略该配置；非 nil 时会覆盖构造级 Reranker。
func WithInputReranker(reranker Reranker) InputOption {
	return func(req *Input) {
		if reranker != nil {
			req.reranker = reranker
		}
	}
}

// WithInputRerankEnabled 设置单次请求是否启用重排。
//
// false 会跳过构造级或请求级 reranker，true 会在存在 reranker 时启用重排。
func WithInputRerankEnabled(enabled bool) InputOption {
	return func(req *Input) {
		req.rerankEnabled = &enabled
	}
}

// WithInputRerankWindowSize 设置单次请求进入重排的候选窗口大小。
//
// windowSize 小于等于 0 时忽略该配置，继续使用构造级配置或默认窗口。
func WithInputRerankWindowSize(windowSize int) InputOption {
	return func(req *Input) {
		if windowSize > 0 {
			req.rerankWindow = windowSize
		}
	}
}

// WithInputRerankTopK 设置单次请求传给重排器的 topN。
//
// topK 小于等于 0 时忽略该配置，继续使用构造级配置或最终返回 topK。
func WithInputRerankTopK(topK int) InputOption {
	return func(req *Input) {
		if topK > 0 {
			req.rerankTopK = topK
		}
	}
}

// WithInputRerankFailOpen 设置单次请求的重排失败回退策略。
//
// true 表示 reranker 返回错误时回退第一阶段融合排序；false 表示直接返回错误。
func WithInputRerankFailOpen(enabled bool) InputOption {
	return func(req *Input) {
		req.rerankFailOpen = &enabled
	}
}

// WithInputRerankMinScore 设置单次请求的最低重排分数。
//
// 设置后，低于该分数的 rerank 结果会被过滤；分数尺度由具体 reranker 决定。
func WithInputRerankMinScore(score float64) InputOption {
	return func(req *Input) {
		req.rerankMinScore = &score
	}
}

// WithInputRerankDocumentBuilder 设置单次请求的重排文档构造器。
//
// builder 为 nil 时忽略该配置，继续使用构造级配置或默认 content 字段。
func WithInputRerankDocumentBuilder(builder RerankDocumentBuilder) InputOption {
	return func(req *Input) {
		if builder != nil {
			req.rerankBuilder = builder
		}
	}
}

// Output 表示一次检索命中的通用结果。
//
// ID 是 ES hit 的 _id，Content 优先使用 parent chunk 内容，Score 是第一阶段融合分。
// RerankScore 仅在重排成功且该结果有重排分数时填充，Source 是调用方 decoder 的输出。
type Output[T any] struct {
	ID          string
	Content     string
	Score       float64
	RerankScore *float64
	Source      T
}
