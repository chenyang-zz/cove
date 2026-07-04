package tool

import "context"

// Descriptor 描述一个可以暴露给模型或编排器选择的工具。
//
// Name 必须非空，并在同一个 Registry 中唯一。InputSchema 和 OutputSchema
// 使用 JSON Schema 兼容的 map 表示，Annotations 用于承载模型或 UI 可选理解的
// 附加元数据。
type Descriptor struct {
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	InputSchema  map[string]any `json:"input_schema,omitempty"`
	OutputSchema map[string]any `json:"output_schema,omitempty"`
	Annotations  map[string]any `json:"annotations,omitempty"`
}

// Input 表示传给工具的通用结构化参数。
type Input map[string]any

// Output 表示工具调用结果。
//
// Text 是最常见的文本观察结果。Parts 用于表达图片、文件或多段文本等结构化结果。
// Metadata 用于返回调用统计、错误类型或调用方需要保留的非展示字段。
type Output struct {
	Text     string         `json:"text,omitempty"`
	Parts    []Part         `json:"parts,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Part 表示工具输出中的一个结构化片段。
//
// Type 由调用方约定，常见值可以是 text、image、file。Text 用于文本片段，
// Data 用于二进制片段，MIME 描述 Data 的媒体类型。
type Part struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
	Data []byte `json:"data,omitempty"`
	MIME string `json:"mime,omitempty"`
}

// Tool 表示一个可被模型或编排器调用的工具。
//
// Describe 返回稳定的工具元信息。Invoke 执行工具调用；实现应尊重 ctx 的取消信号，
// 并把业务依赖通过构造函数、闭包或 ctx 注入，而不是依赖全局状态。
type Tool interface {
	Describe(ctx context.Context) (Descriptor, error)
	Invoke(ctx context.Context, input Input) (Output, error)
}

// InvokeFunc 是 FuncTool 使用的函数签名。
type InvokeFunc func(ctx context.Context, input Input) (Output, error)
