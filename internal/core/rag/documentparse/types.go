package documentparse

import "context"

// Input 表示待解析文档的原始内容和文件扩展名。
type Input struct {
	Data    []byte
	FileExt string
}

// Output 表示文档解析后的纯文本和归一化文件扩展名。
type Output struct {
	Text    string
	FileExt string
}

// Extractor 定义单一文件格式的文本提取行为。
type Extractor interface {
	Extract(ctx context.Context, input Input) (string, error)
}

// TextDecoder 定义纯文本字节到字符串的解码行为。
type TextDecoder interface {
	Decode(data []byte) (string, error)
}

// extractorFunc 把函数适配为 Extractor。
type extractorFunc func(ctx context.Context, input Input) (string, error)

// Extract 调用底层函数提取文本。
func (fn extractorFunc) Extract(ctx context.Context, input Input) (string, error) {
	return fn(ctx, input)
}
