package channel

import (
	"context"
	"testing"
)

type fakeProvider struct{ descriptor ProviderDescriptor }

func (f fakeProvider) Descriptor() ProviderDescriptor                           { return f.descriptor }
func (fakeProvider) Receive(context.Context, AccountConfig, EventHandler) error { return nil }
func (fakeProvider) Send(context.Context, AccountConfig, OutboundMessage) (Receipt, error) {
	return Receipt{State: DeliverySent}, nil
}
func (fakeProvider) SetTyping(context.Context, AccountConfig, Route, bool) error { return nil }

func validFakeProvider(name ProviderName) fakeProvider {
	return fakeProvider{descriptor: ProviderDescriptor{
		Name: name, DisplayName: string(name), MaxTextLength: 100,
		Capabilities: Capabilities{OutboundText: true},
	}}
}

// 测试注册表拒绝同名 Provider 覆盖，避免账号被错误适配器接管。
func TestRegistryRejectsDuplicateProvider(t *testing.T) {
	registry, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(validFakeProvider(ProviderTelegram)); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(validFakeProvider(ProviderTelegram)); err == nil {
		t.Fatal("expected duplicate provider error")
	}
}

// 测试描述校验要求出站文本能力和有效限长。
func TestProviderDescriptorValidateCapabilities(t *testing.T) {
	descriptor := ProviderDescriptor{Name: ProviderWebhook, DisplayName: "Webhook", MaxTextLength: 0}
	if err := descriptor.Validate(); err == nil {
		t.Fatal("expected invalid capabilities")
	}
	descriptor.Capabilities.OutboundText = true
	descriptor.MaxTextLength = 4096
	if err := descriptor.Validate(); err != nil {
		t.Fatalf("expected valid descriptor: %v", err)
	}
}

// 测试路由键同时隔离账号、聊天和线程。
func TestStableRouteKeySeparatesThread(t *testing.T) {
	base := StableRouteKey("account", "chat", "thread-a")
	if base == StableRouteKey("account", "chat", "thread-b") {
		t.Fatal("different threads must not share route key")
	}
	if base != StableRouteKey("account", "chat", "thread-a") {
		t.Fatal("route key must be deterministic")
	}
}

// 测试文本切分保持 Unicode 字符完整并遵循平台限长。
func TestSplitTextPreservesUnicode(t *testing.T) {
	parts := SplitText("你好世界\n再次见面", 4)
	if len(parts) != 3 || parts[0] != "你好世界" || parts[1] != "再次见" || parts[2] != "面" {
		t.Fatalf("unexpected parts: %#v", parts)
	}
}
