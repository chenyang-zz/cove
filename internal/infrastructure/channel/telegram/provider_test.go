package telegram

import (
	"errors"
	"testing"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	tgbot "github.com/go-telegram/bot"
)

// 测试 Telegram 群聊提及匹配忽略大小写和可选 @ 前缀。
func TestContainsMention(t *testing.T) {
	if !containsMention("hi @CoveBot", "@covebot") {
		t.Fatal("expected mention")
	}
	if containsMention("hi everyone", "covebot") {
		t.Fatal("unexpected mention")
	}
}

// TestTelegramDeliveryStateAvoidsBlindRetry 验证网络不确定错误停止重发，而明确限流允许安全重试。
func TestTelegramDeliveryStateAvoidsBlindRetry(t *testing.T) {
	if got := telegramDeliveryState(errors.New("connection reset"), 0); got != corechannel.DeliveryUnknown {
		t.Fatalf("network error state = %q", got)
	}
	if got := telegramDeliveryState(&tgbot.TooManyRequestsError{RetryAfter: 1}, 0); got != corechannel.DeliveryTemporary {
		t.Fatalf("rate limit state = %q", got)
	}
	if got := telegramDeliveryState(tgbot.ErrorForbidden, 0); got != corechannel.DeliveryPermanent {
		t.Fatalf("forbidden state = %q", got)
	}
}
