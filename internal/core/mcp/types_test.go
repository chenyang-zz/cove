package mcp

import (
	"bytes"
	"testing"
)

// TestContentFromJSONPreservesProtocolContent 验证 MCP 文本、资源和媒体内容会保留原始协议字段及解码后的数据。
func TestContentFromJSONPreservesProtocolContent(t *testing.T) {
	resource, err := contentFromJSON([]byte(`{"type":"resource","resource":{"uri":"file:///report.txt","text":"full body"}}`))
	if err != nil {
		t.Fatalf("contentFromJSON resource error = %v, want nil", err)
	}
	if resource.Type != "resource" || resource.URI != "file:///report.txt" || !bytes.Contains(resource.Raw, []byte("full body")) {
		t.Fatalf("resource content = %#v, want URI and full raw body", resource)
	}

	image, err := contentFromJSON([]byte(`{"type":"image","data":"AQID","mimeType":"image/png"}`))
	if err != nil {
		t.Fatalf("contentFromJSON image error = %v, want nil", err)
	}
	if image.MIMEType != "image/png" || !bytes.Equal(image.Data, []byte{1, 2, 3}) || len(image.Raw) == 0 {
		t.Fatalf("image content = %#v, want decoded bytes and raw payload", image)
	}
}
