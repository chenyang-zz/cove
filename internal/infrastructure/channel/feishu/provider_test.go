package feishu

import "testing"

// 测试飞书发送者 ID 按 Open ID、User ID、Union ID 顺序回退。
func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "user-id", "union-id"); got != "user-id" {
		t.Fatalf("got %q", got)
	}
}
