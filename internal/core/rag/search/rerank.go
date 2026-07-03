package search

import (
	"context"

	"github.com/boxify/api-go/internal/core/valuex"
)

// rerankConfig 是构造级和请求级重排配置合并后的内部快照。
type rerankConfig struct {
	enabled    bool
	reranker   Reranker
	windowSize int
	topK       int
	failOpen   bool
	minScore   *float64
	builder    RerankDocumentBuilder
}

// applyRerank 对第一阶段融合候选执行可选重排。
//
// 返回的 id 顺序是最终候选顺序，score map 只包含 reranker 返回的分数。
// 当重排未启用或 fail-open 触发时，会返回原始候选顺序和 nil score map。
func (s *Searcher[T]) applyRerank(ctx context.Context, req Input, candidateIDs []string, hits map[string]map[string]any, topK int, recallSize int) ([]string, map[string]float64, error) {
	config := s.resolveRerankConfig(req, topK, recallSize)
	if !config.enabled || config.reranker == nil || len(candidateIDs) == 0 {
		return candidateIDs, nil, nil
	}

	// 只把第一阶段融合排序后的前 windowSize 个候选交给 reranker，控制成本和延迟。
	windowIDs := candidateIDs
	if len(windowIDs) > config.windowSize {
		windowIDs = windowIDs[:config.windowSize]
	}

	// 文档文本由 builder 决定，core 包不理解标题、摘要等业务字段。
	documents := make([]string, 0, len(windowIDs))
	for _, id := range windowIDs {
		documents = append(documents, config.builder(hits[id]))
	}

	reranked, err := config.reranker.Rerank(ctx, req.Query, documents, config.topK)
	if err != nil {
		// rerank 是可选增强；fail-open 时保留第一阶段融合结果，避免辅助服务阻断检索。
		if config.failOpen {
			return candidateIDs, nil, nil
		}
		return nil, nil, err
	}
	ids, scores := mapRerankResults(windowIDs, reranked, config.minScore)
	return ids, scores, nil
}

// resolveRerankConfig 合并构造级和请求级重排配置。
//
// 请求级配置优先于构造级配置；未显式配置窗口、topK 或 builder 时使用稳定默认值。
func (s *Searcher[T]) resolveRerankConfig(req Input, topK int, recallSize int) rerankConfig {
	config := rerankConfig{
		enabled:    true,
		reranker:   s.Reranker,
		windowSize: s.RerankWindowSize,
		topK:       s.RerankTopK,
		failOpen:   s.RerankFailOpen,
		minScore:   s.RerankMinScore,
		builder:    s.RerankDocumentBuilder,
	}
	if req.reranker != nil {
		config.reranker = req.reranker
	}
	if req.rerankEnabled != nil {
		config.enabled = *req.rerankEnabled
	}
	if req.rerankWindow > 0 {
		config.windowSize = req.rerankWindow
	}
	if req.rerankTopK > 0 {
		config.topK = req.rerankTopK
	}
	if req.rerankFailOpen != nil {
		config.failOpen = *req.rerankFailOpen
	}
	if req.rerankMinScore != nil {
		config.minScore = req.rerankMinScore
	}
	if req.rerankBuilder != nil {
		config.builder = req.rerankBuilder
	}

	// 默认窗口覆盖最终返回数量和召回池大小，确保 reranker 能看到足够候选。
	if config.windowSize <= 0 {
		config.windowSize = max(topK, recallSize)
	}
	if config.topK <= 0 {
		config.topK = topK
	}
	if config.builder == nil {
		config.builder = defaultRerankDocumentBuilder
	}
	return config
}

// defaultRerankDocumentBuilder 默认使用 content 字段作为重排文本。
func defaultRerankDocumentBuilder(src map[string]any) string {
	return valuex.String(src["content"])
}

// mapRerankResults 把 reranker 返回的下标映射回候选 id，并过滤不可用结果。
func mapRerankResults(candidateIDs []string, reranked []RerankResult, minScore *float64) ([]string, map[string]float64) {
	ids := make([]string, 0, len(reranked))
	scores := make(map[string]float64, len(reranked))
	seen := map[int]struct{}{}
	for _, item := range reranked {
		// 外部 reranker 结果不可信，越界、重复或低分候选都在 core 层过滤掉。
		if item.Index < 0 || item.Index >= len(candidateIDs) {
			continue
		}
		if _, ok := seen[item.Index]; ok {
			continue
		}
		if minScore != nil && item.Score < *minScore {
			continue
		}
		seen[item.Index] = struct{}{}
		id := candidateIDs[item.Index]
		ids = append(ids, id)
		scores[id] = item.Score
	}
	return ids, scores
}
