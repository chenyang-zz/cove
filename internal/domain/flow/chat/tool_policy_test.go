package chat

import "testing"

// 测试未知工具策略安全回退到继承模式。
func TestNormalizeToolPolicy(t *testing.T) {
	if got := normalizeToolPolicy(" SAFE "); got != ToolPolicySafe {
		t.Fatalf("got %q", got)
	}
	if got := normalizeToolPolicy("unexpected"); got != ToolPolicyInherit {
		t.Fatalf("got %q", got)
	}
}
