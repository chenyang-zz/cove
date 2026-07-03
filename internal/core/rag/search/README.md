# core/rag/search Rerank 设计

## Summary

`core/rag/search` 采用“第一阶段召回 + 第二阶段可选重排”的通用 RAG 检索结构：先执行向量召回和 BM25 召回，再做分数归一化与融合，随后可选调用 `Reranker` 对候选窗口进行重排，最后输出结果。

该机制保持 core 包业务无关：不绑定 Cohere、Jina、ES Inference、本地 cross-encoder 等具体供应商，只定义可注入接口和通用流程。Rerank 默认是增强能力，失败默认 fail-open 回退融合排序，避免辅助能力阻断主检索链路。

## References

- Cohere 将 rerank 定位为 keyword/vector 第一阶段检索后的第二阶段语义增强，适合 RAG 场景：[Cohere Reranking](https://docs.cohere.com/docs/reranking-with-cohere)。
- LangChain 用 `ContextualCompressionRetriever` 包装 base retriever，再接 `CohereRerank`：[LangChain Cohere reranker integration](https://python.langchain.com/docs/integrations/retrievers/cohere-reranker/)。
- LlamaIndex 的 `CohereRerank` / `SentenceTransformerRerank` 作为 node postprocessor，对 nodes 重排并返回 top N：[LlamaIndex Node Postprocessors](https://docs.llamaindex.ai/en/stable/module_guides/querying/node_postprocessors/node_postprocessors/)。
- Haystack 明确区分 retriever `top_k` 和 ranker `top_k`，ranker 通常接在 retriever 后，且 ranker topK 可更小：[Haystack TransformersSimilarityRanker](https://docs.haystack.deepset.ai/docs/transformerssimilarityranker)。
- Elasticsearch 的 `text_similarity_reranker` 使用 nested retriever 产出第一阶段结果，再用 `rank_window_size` 控制进入 rerank 的候选窗口：[Elasticsearch retriever API](https://www.elastic.co/guide/en/elasticsearch/reference/current/retriever.html)。

## Search Pipeline

目标流水线：

```text
query
  -> build filters
  -> embed query
  -> vector recall
  -> BM25 recall
  -> optional vector score gate
  -> normalize + weighted fusion
  -> optional rerank
  -> topK trim
  -> parent content fallback
  -> source decode
  -> outputs
```

Rerank 只处理融合后的候选窗口，不替代向量/BM25 融合逻辑。`MinVectorScore` 仍只负责过滤低向量相关度候选，过滤后继续参与融合和 rerank。

## Public API

保留现有重排接口：

```go
type Reranker interface {
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
}

type RerankResult struct {
	Index int
	Score float64
}
```

新增通用文档构造接口：

```go
type RerankDocumentBuilder func(src map[string]any) string
```

`Output[T]` 增加可选 rerank 分数：

```go
type Output[T any] struct {
	ID          string
	Content     string
	Score       float64
	RerankScore *float64
	Source      T
}
```

`Score` 继续表示第一阶段融合分数，`RerankScore` 仅表示第二阶段重排分数，避免混用不同分数体系。

## Configuration

构造级配置：

```go
WithReranker(reranker Reranker)
WithRerankWindowSize(n int)
WithRerankTopK(n int)
WithRerankFailOpen(enabled bool)
WithRerankMinScore(score float64)
WithRerankDocumentBuilder(builder RerankDocumentBuilder)
```

请求级配置：

```go
WithInputReranker(reranker Reranker)
WithInputRerankEnabled(enabled bool)
WithInputRerankWindowSize(n int)
WithInputRerankTopK(n int)
WithInputRerankFailOpen(enabled bool)
WithInputRerankMinScore(score float64)
WithInputRerankDocumentBuilder(builder RerankDocumentBuilder)
```

配置优先级：

1. 请求级配置优先。
2. 构造级配置次之。
3. 未配置 reranker 或显式 disabled 时跳过 rerank。
4. 默认 `failOpen=true`。
5. 默认 rerank window 使用 `max(topK, recallSize)`。
6. 默认 rerank topK 使用最终 `topK`。
7. 默认 document builder 读取 `_source["content"]`。

## Implementation Structure

`rerank.go` 专门放 rerank 机制实现：

- `rerankConfig`。
- `resolveRerankConfig`。
- `Searcher.applyRerank`。
- 默认 `defaultRerankDocumentBuilder`。
- rerank 结果去重、越界过滤、min score 过滤、score map 构造 helper。

`search.go` 保持主流程清晰：

```go
candidateIDs := rankedIDs(fused, max(topK, recallSize))

candidateIDs, rerankScores, err := s.applyRerank(ctx, req, candidateIDs, hits, topK, recallSize)
if err != nil {
	return nil, err
}

if len(candidateIDs) > topK {
	candidateIDs = candidateIDs[:topK]
}
return s.resultsForIDs(ctx, candidateIDs, hits, fused, rerankScores)
```

公开类型继续放在 `types.go`，长期 option 放 `options.go`，请求级 option 放 `types.go`，避免把公开 API 分散到实现文件。

## Failure Behavior

- reranker 返回错误：
  - `failOpen=true`：回退融合排序，不返回错误。
  - `failOpen=false`：返回 rerank 错误。
- reranker 返回越界 index：忽略。
- reranker 返回重复 index：只保留第一次。
- reranker 返回少于 topK：按实际 rerank 结果返回，不自动补融合候选。
- reranker 返回空结果：视为 rerank 成功但无结果；若需要回退，应由 adapter 返回错误或后续显式增加策略。
- `RerankMinScore` 设置后，只保留 rerank score 达标结果。
