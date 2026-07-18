package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
)

// 测试 Webhook 正确签名可通过校验。
func TestVerifyAcceptsValidSignature(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	timestamp := "1700000000"
	body := []byte(`{"event_id":"e-1"}`)
	if err := Verify("secret", timestamp, Sign("secret", timestamp, body), body, now, time.Minute); err != nil {
		t.Fatalf("verify valid signature: %v", err)
	}
}

// 测试 Webhook 拒绝时间窗口外的重放请求。
func TestVerifyRejectsExpiredTimestamp(t *testing.T) {
	now := time.Unix(1_700_000_600, 0)
	timestamp := "1700000000"
	body := []byte(`{}`)
	if err := Verify("secret", timestamp, Sign("secret", timestamp, body), body, now, time.Minute); err == nil {
		t.Fatal("expected expired timestamp error")
	}
}

// 测试 Webhook 媒体域名白名单允许子域但拒绝相似后缀和 IP 字面量。
func TestHostAllowedRequiresDomainBoundary(t *testing.T) {
	allowlist := []string{"media.example.com"}
	if !hostAllowed("cdn.media.example.com", allowlist) {
		t.Fatal("expected subdomain to be allowed")
	}
	if hostAllowed("media.example.com.evil.test", allowlist) || hostAllowed("127.0.0.1", allowlist) {
		t.Fatal("expected deceptive host and IP to be denied")
	}
}

// TestSendSignsCallbackAndCarriesDeliveryID 验证回调签名、幂等键和最终文本协议完整一致。
func TestSendSignsCallbackAndCarriesDeliveryID(t *testing.T) {
	const secret = "callback-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := Verify(secret, r.Header.Get("X-Cove-Timestamp"), r.Header.Get("X-Cove-Signature"), body, time.Now(), time.Minute); err != nil {
			t.Errorf("invalid callback signature: %v", err)
		}
		var payload struct {
			DeliveryID string `json:"delivery_id"`
			Text       string `json:"text"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Error(err)
		}
		if payload.DeliveryID != "delivery-1" || r.Header.Get("X-Cove-Delivery-ID") != "delivery-1" || payload.Text != "hello" {
			t.Errorf("unexpected callback payload or headers: %#v", payload)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	provider := New(server.Client())
	receipt, err := provider.Send(context.Background(), corechannel.AccountConfig{
		Credentials: map[string]string{"signing_secret": secret}, Settings: map[string]any{"callback_url": server.URL},
	}, corechannel.OutboundMessage{DeliveryID: "delivery-1", Route: corechannel.Route{ChatID: "chat"}, Text: "hello"})
	if err != nil || receipt.State != corechannel.DeliverySent {
		t.Fatalf("send receipt = %#v, err = %v", receipt, err)
	}
}
