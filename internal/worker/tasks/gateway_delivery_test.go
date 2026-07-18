package tasks

import (
	"errors"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/models"
)

// TestDecodeOutboxReplyPreservesPlatformMessageID 验证最终回复可映射回入站平台消息或线程。
func TestDecodeOutboxReplyPreservesPlatformMessageID(t *testing.T) {
	reply, err := decodeOutboxReply(models.JSONMap{"message_id": "platform-message-1"})
	if err != nil {
		t.Fatal(err)
	}
	if reply == nil || reply.MessageID != "platform-message-1" {
		t.Fatalf("unexpected reply reference: %#v", reply)
	}
}

// TestDecodeOutboxReplyAllowsNoReference 验证没有平台消息 ID 时正常降级为普通发送。
func TestDecodeOutboxReplyAllowsNoReference(t *testing.T) {
	reply, err := decodeOutboxReply(nil)
	if err != nil || reply != nil {
		t.Fatalf("reply = %#v, err = %v", reply, err)
	}
}

// TestGatewayTurnErrorKindDoesNotExposeSensitiveError 验证回合日志只记录错误分类，不泄露底层敏感错误文本。
func TestGatewayTurnErrorKindDoesNotExposeSensitiveError(t *testing.T) {
	sensitive := "token=secret media_url=https://private.example/file"
	got := gatewayTurnErrorKind(errors.New(sensitive))
	if got != "internal" {
		t.Fatalf("gatewayTurnErrorKind() = %q, want internal", got)
	}
	if strings.Contains(got, sensitive) || strings.Contains(got, "secret") {
		t.Fatalf("gatewayTurnErrorKind() exposed sensitive error text: %q", got)
	}
}

// TestGatewayDeliverErrorKindDoesNotExposeSensitiveError 验证投递日志只记录错误分类，不泄露渠道响应或回调地址。
func TestGatewayDeliverErrorKindDoesNotExposeSensitiveError(t *testing.T) {
	sensitive := "provider_response=secret callback_url=https://private.example/hook"
	got := gatewayDeliverErrorKind(errors.New(sensitive))
	if got != "internal" {
		t.Fatalf("gatewayDeliverErrorKind() = %q, want internal", got)
	}
	if strings.Contains(got, sensitive) || strings.Contains(got, "secret") {
		t.Fatalf("gatewayDeliverErrorKind() exposed sensitive error text: %q", got)
	}
}
