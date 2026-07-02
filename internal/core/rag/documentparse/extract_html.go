package documentparse

import (
	"bytes"
	"context"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// extractHTML 使用 goquery 解析 HTML，并提取可用于检索的正文文本。
func extractHTML(ctx context.Context, input Input) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(input.Data))
	if err != nil {
		return "", err
	}

	// 这些节点通常不是正文内容，先从 DOM 中移除再取 Text，避免污染 RAG 文本。
	doc.Find("head,script,style,noscript,template,svg").Remove()
	return strings.TrimSpace(doc.Text()), nil
}
