package domain

import "testing"

func TestDocumentStatusConstants(t *testing.T) {
	// 验证文档状态常量集中由 domain 层提供，避免 logic/worker 重复定义字符串。
	tests := map[string]string{
		"pending": DocumentStatusPending,
		"parsing": DocumentStatusParsing,
		"done":    DocumentStatusDone,
		"failed":  DocumentStatusFailed,
	}
	for want, got := range tests {
		if got != want {
			t.Fatalf("document status = %q, want %q", got, want)
		}
	}
}
