package documentparse

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

// extractPDF 使用 PDF 解析库读取文本层，不处理扫描件或 OCR。
func extractPDF(ctx context.Context, input Input) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	reader := bytes.NewReader(input.Data)
	pdfReader, err := pdf.NewReader(reader, int64(len(input.Data)))
	if err != nil {
		return "", fmt.Errorf("parse pdf failed: %w", err)
	}

	// GetPlainText 返回 PDF 文本层 reader；扫描件或无文本层会在统一出口被判为空文本。
	textReader, err := pdfReader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract pdf text failed: %w", err)
	}
	text, err := io.ReadAll(textReader)
	if err != nil {
		return "", fmt.Errorf("read pdf text failed: %w", err)
	}
	return string(text), nil
}
