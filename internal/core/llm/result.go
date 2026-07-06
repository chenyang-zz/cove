package llm

import coretool "github.com/boxify/api-go/internal/core/tool"

// LLMResult 表示一次非流式 LLM 生成调用的结构化结果。
//
// Text 保存模型返回的文本内容。ToolCalls 保存模型请求执行的工具调用。Usage 保存跨供应商
// 归一化后的 token 用量。RawJSON 保留供应商原始响应 JSON，便于审计和排查兼容问题。
type LLMResult struct {
	Text       string
	ToolCalls  []LLMToolCall
	Model      string
	Provider   string
	ID         string
	StopReason string
	Usage      TokenUsage
	RawJSON    string
	Metadata   map[string]any
}

// LLMToolCall 表示模型生成的一次工具调用请求。
//
// Input 是从 RawInput 解析出的 JSON object。解析失败时 Input 为空，RawInput 仍保留
// 模型原始参数字符串，调用方可以自行记录或兜底处理。
type LLMToolCall struct {
	ID       string
	Name     string
	Input    coretool.Input
	RawInput string
}

// TokenUsage 表示跨供应商归一化后的 token 用量。
type TokenUsage struct {
	InputTokens              int64
	OutputTokens             int64
	TotalTokens              int64
	CacheCreationInputTokens int64
	CacheReadInputTokens     int64
}
