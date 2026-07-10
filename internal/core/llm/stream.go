package llm

import "context"

// StreamEventKind 表示模型流中的事件类别。
type StreamEventKind string

const (
	// StreamEventTextDelta 表示可展示的模型文本增量。
	StreamEventTextDelta StreamEventKind = "text_delta"
	// StreamEventToolCall 表示已经聚合完成的一次原生工具调用。
	StreamEventToolCall StreamEventKind = "tool_call"
	// StreamEventDone 表示供应商正常结束本次流。
	StreamEventDone StreamEventKind = "done"
	// StreamEventError 表示流在建立成功后的运行期错误。
	StreamEventError StreamEventKind = "error"
)

// StreamEvent 表示供应商无关的模型流事件。
//
// Text 仅在 StreamEventTextDelta 时有值，ToolCall 仅在 StreamEventToolCall 时有值，
// Err 仅在 StreamEventError 时有值。调用方应持续读取到 Done 或 Error，而不是仅依赖
// 通道关闭判断请求是否成功。
type StreamEvent struct {
	Kind     StreamEventKind
	Text     string
	ToolCall *LLMToolCall
	Err      error
}

// StreamEventClient 表示支持结构化流式响应的模型客户端。
//
// StreamEvents 在请求建立失败时直接返回 error；建立成功后的中断和供应商错误通过
// StreamEventError 传递，随后通道关闭。
type StreamEventClient interface {
	StreamEvents(ctx context.Context, messages []*Message, opts ...ModelCallOption) (<-chan StreamEvent, error)
}

// ToolStreamEventClient 表示支持原生工具调用流的模型客户端。
//
// StreamWithTools 会实时发送文本增量，并在工具参数聚合完整后发送 StreamEventToolCall。
type ToolStreamEventClient interface {
	StreamWithTools(ctx context.Context, messages []*Message, opts ...ModelCallOption) (<-chan StreamEvent, error)
}
