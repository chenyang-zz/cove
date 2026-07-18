package gateway

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/models"
	"github.com/google/uuid"
)

// 测试外部文本和回复引用会被明确包装为不可信内容。
func TestUntrustedMessageWrapsExternalContent(t *testing.T) {
	got := untrustedMessage(corechannel.InboundEvent{Text: "hello", Reply: &corechannel.ReplyReference{Text: "old"}})
	if !strings.Contains(got, "<external_untrusted>") || !strings.Contains(got, "Reply reference") {
		t.Fatalf("unexpected wrapper: %q", got)
	}
}

// 测试配对码固定六位、一小时过期并只保存哈希。
func TestPairingCodeUsesHashAndExpiry(t *testing.T) {
	code, hash, expiresAt, err := pairingCode(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 || len(hash) != 64 || hash == code {
		t.Fatalf("unexpected pairing values: %q %q", code, hash)
	}
	if time.Until(expiresAt) < 59*time.Minute {
		t.Fatalf("unexpected expiry: %v", expiresAt)
	}
}

// TestInboundUserMessageIDIsDeterministic 验证同一 Inbox 恢复时复用用户消息 ID 且不与 Assistant ID 冲突。
func TestInboundUserMessageIDIsDeterministic(t *testing.T) {
	inboxID := uuid.New()
	first := inboundUserMessageID(inboxID)
	second := inboundUserMessageID(inboxID)
	assistant := uuid.NewSHA1(uuid.NameSpaceOID, inboxID[:])
	if first != second || first == uuid.Nil || first == assistant {
		t.Fatalf("unexpected deterministic IDs: user=%s/%s assistant=%s", first, second, assistant)
	}
}

// TestHasExtractedAttachment 验证纯媒体恢复只在存在可用提取文本时进入 Agent。
func TestHasExtractedAttachment(t *testing.T) {
	if hasExtractedAttachment(&models.MessageMetaData{Attachments: []models.MessageAttachmentMeta{{ParseError: "failed"}}}) {
		t.Fatal("parse error metadata must not be treated as extracted content")
	}
	if !hasExtractedAttachment(&models.MessageMetaData{Attachments: []models.MessageAttachmentMeta{{ExtractedText: "OCR"}}}) {
		t.Fatal("expected extracted attachment content")
	}
}

// TestGatewayEnqueueErrorIsRecoverable 验证包装后的队列错误仍可被 Inbox 状态机识别并对账恢复。
func TestGatewayEnqueueErrorIsRecoverable(t *testing.T) {
	err := fmt.Errorf("%w: redis unavailable", errGatewayEnqueue)
	if !errors.Is(err, errGatewayEnqueue) {
		t.Fatal("expected wrapped enqueue error to remain identifiable")
	}
}

// TestSelectedAgentConfigIDPrefersBinding 验证 AgentConfig 优先级为绑定覆盖高于账号默认。
func TestSelectedAgentConfigIDPrefersBinding(t *testing.T) {
	accountID := uuid.New()
	bindingID := uuid.New()
	got := selectedAgentConfigID(
		&models.ChannelAccount{DefaultAgentConfigID: &accountID},
		&models.ChannelBinding{AgentConfigID: &bindingID},
	)
	if got == nil || *got != bindingID {
		t.Fatalf("selected agent config = %v, want binding %s", got, bindingID)
	}
}
