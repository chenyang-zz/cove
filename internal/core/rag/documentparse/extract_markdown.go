package documentparse

import (
	"bytes"
	"context"

	"github.com/yuin/goldmark"
)

// extractMarkdown 使用 goldmark 把 Markdown 转为 HTML 后复用 HTML 文本提取。
func extractMarkdown(ctx context.Context, input Input) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	text, err := utf8Decoder{}.Decode(input.Data)
	if err != nil {
		return "", err
	}

	// Markdown 解析交给 CommonMark 兼容实现，再由 HTML 提取器去掉标签和链接地址。
	var html bytes.Buffer
	if err := goldmark.Convert([]byte(text), &html); err != nil {
		return "", err
	}
	return extractHTML(ctx, Input{Data: html.Bytes(), FileExt: ".html"})
}
