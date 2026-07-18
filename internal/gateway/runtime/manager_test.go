package runtime

import (
	"testing"
	"time"
)

// 测试网关时长配置仅接受正值并提供稳定回退。
func TestParsePositiveDuration(t *testing.T) {
	if got := parsePositiveDuration("5s", time.Minute); got != 5*time.Second {
		t.Fatalf("got %v", got)
	}
	if got := parsePositiveDuration("invalid", time.Minute); got != time.Minute {
		t.Fatalf("got %v", got)
	}
}
