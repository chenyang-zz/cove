package search

const (
	defaultIndex          = "cove_chunks"
	defaultVectorWeight   = 0.6
	defaultBM25Weight     = 0.4
	defaultTopK           = 5
	defaultRecallSize     = 20
	defaultEmbeddingDim   = 1024
	defaultRerankFailOpen = true
)

// Options 定义 Searcher 的长期配置。
//
// FilterBuilder 和 sourceDecoder 用于把业务过滤和业务元数据解码留给调用方。
// Rerank 相关字段只控制第二阶段重排，未配置 Reranker 时不会触发重排。
type Options struct {
	Index                 string
	EmbeddingDim          int
	RecallSize            int
	VectorWeight          float64
	BM25Weight            float64
	KnnOversample         int
	Embedder              Embedder
	Reranker              Reranker
	RerankWindowSize      int
	RerankTopK            int
	RerankFailOpen        bool
	RerankMinScore        *float64
	RerankDocumentBuilder RerankDocumentBuilder
	FilterBuilder         FilterBuilder
	sourceDecoder         any
}

// Option 修改 Searcher 的长期配置。
type Option func(*Options)

// WithIndex 设置 Elasticsearch 索引名。
func WithIndex(index string) Option {
	return func(opts *Options) {
		if index != "" {
			opts.Index = index
		}
	}
}

// WithEmbeddingDim 设置向量化维度。
func WithEmbeddingDim(embeddingDim int) Option {
	return func(opts *Options) {
		if embeddingDim > 0 {
			opts.EmbeddingDim = embeddingDim
		}
	}
}

// WithRecallSize 设置默认召回池大小。
func WithRecallSize(recallSize int) Option {
	return func(opts *Options) {
		if recallSize > 0 {
			opts.RecallSize = recallSize
		}
	}
}

// WithVectorWeight 设置向量召回分数融合权重。
func WithVectorWeight(vectorWeight float64) Option {
	return func(opts *Options) {
		opts.VectorWeight = vectorWeight
	}
}

// WithBM25Weight 设置 BM25 召回分数融合权重。
func WithBM25Weight(bm25Weight float64) Option {
	return func(opts *Options) {
		opts.BM25Weight = bm25Weight
	}
}

// WithKnnOversample 设置 ES knn num_candidates 的过采样倍数。
//
// knnOversample 小于等于 0 时不写 num_candidates，由 ES 使用默认策略。
func WithKnnOversample(knnOversample int) Option {
	return func(opts *Options) {
		if knnOversample > 0 {
			opts.KnnOversample = knnOversample
		}
	}
}

// WithEmbedder 设置默认向量化客户端。
func WithEmbedder(embedder Embedder) Option {
	return func(opts *Options) {
		if embedder != nil {
			opts.Embedder = embedder
		}
	}
}

// WithReranker 设置默认重排器。
//
// reranker 为 nil 时等价于关闭构造级重排；请求级 WithInputReranker 仍可覆盖。
func WithReranker(reranker Reranker) Option {
	return func(opts *Options) {
		opts.Reranker = reranker
	}
}

// WithRerankWindowSize 设置进入重排的候选窗口大小。
//
// windowSize 小于等于 0 时忽略该配置，默认使用 max(topK, recallSize)。
func WithRerankWindowSize(windowSize int) Option {
	return func(opts *Options) {
		if windowSize > 0 {
			opts.RerankWindowSize = windowSize
		}
	}
}

// WithRerankTopK 设置传给重排器的 topN。
//
// topK 小于等于 0 时忽略该配置，默认使用最终返回 topK。
func WithRerankTopK(topK int) Option {
	return func(opts *Options) {
		if topK > 0 {
			opts.RerankTopK = topK
		}
	}
}

// WithRerankFailOpen 设置重排失败时是否回退融合排序。
//
// 默认值为 true；设置为 false 后，reranker 返回错误会让 Search 返回该错误。
func WithRerankFailOpen(enabled bool) Option {
	return func(opts *Options) {
		opts.RerankFailOpen = enabled
	}
}

// WithRerankMinScore 设置最低重排分数。
//
// 设置后，低于该分数的 rerank 结果会被过滤；分数尺度由具体 reranker 决定。
func WithRerankMinScore(score float64) Option {
	return func(opts *Options) {
		opts.RerankMinScore = &score
	}
}

// WithRerankDocumentBuilder 设置重排文档构造器。
//
// builder 为 nil 时忽略该配置，默认使用 ES _source 中的 content 字段。
func WithRerankDocumentBuilder(builder RerankDocumentBuilder) Option {
	return func(opts *Options) {
		if builder != nil {
			opts.RerankDocumentBuilder = builder
		}
	}
}

// WithFilterBuilder 设置请求过滤构造器。
func WithFilterBuilder(builder FilterBuilder) Option {
	return func(opts *Options) {
		if builder != nil {
			opts.FilterBuilder = builder
		}
	}
}

// WithSourceDecoder 设置 ES _source 到业务元数据的解码器。
func WithSourceDecoder[T any](decoder SourceDecoder[T]) Option {
	return func(opts *Options) {
		if decoder != nil {
			opts.sourceDecoder = decoder
		}
	}
}
