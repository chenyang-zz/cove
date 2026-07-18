package gateway

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// 测试附件 MIME 校验拒绝将 HTML 伪装成图片。
func TestAllowedMediaRejectsMIMEMismatch(t *testing.T) {
	if allowedMedia("image", "image/png", "text/html") {
		t.Fatal("expected mismatched image MIME to be rejected")
	}
	if !allowedMedia("document", "application/pdf", "application/pdf") {
		t.Fatal("expected PDF to be allowed")
	}
}

// TestTruncateExtractedText 验证超长且包含多字节字符的附件文本会按字符安全截断。
func TestTruncateExtractedText(t *testing.T) {
	input := strings.Repeat("中", maxExtractedTextRunes+1)
	got := truncateExtractedText(input)
	if !strings.HasSuffix(got, "[附件提取内容已截断]") {
		t.Fatalf("expected truncation marker, got suffix %q", got[len(got)-30:])
	}
	if !utf8.ValidString(got) {
		t.Fatal("expected valid UTF-8 after truncation")
	}
}
