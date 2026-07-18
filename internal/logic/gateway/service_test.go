package gateway

import (
	"testing"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/infrastructure/security"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/svc"
)

// TestValidateFieldsRejectsMissingAndUnknownCredentials 验证 Provider 凭据声明是账号配置的唯一白名单。
func TestValidateFieldsRejectsMissingAndUnknownCredentials(t *testing.T) {
	fields := []corechannel.FieldDescriptor{{Key: "token", Required: true}}
	if err := validateFields(fields, nil, true); err == nil {
		t.Fatal("expected missing credential to be rejected")
	}
	if err := validateFields(fields, map[string]string{"token": "ok", "extra": "no"}, true); err == nil {
		t.Fatal("expected unknown credential to be rejected")
	}
}

// TestValidateSettingsChecksAllowlistAndURL 验证账号公共设置拒绝未知字段和无效回调地址。
func TestValidateSettingsChecksAllowlistAndURL(t *testing.T) {
	fields := []corechannel.FieldDescriptor{{Key: "callback_url", Type: "url", Required: true}}
	if err := validateSettings(fields, map[string]any{"callback_url": "not-a-url"}, true); err == nil {
		t.Fatal("expected invalid callback URL to be rejected")
	}
	if err := validateSettings(fields, map[string]any{"callback_url": "https://example.com/reply", "extra": true}, true); err == nil {
		t.Fatal("expected unknown setting to be rejected")
	}
	if err := validateSettings(fields, map[string]any{"callback_url": "https://example.com/reply"}, true); err != nil {
		t.Fatalf("expected valid settings, got %v", err)
	}
}

// TestAccountResponseMasksEncryptedCredentials 验证管理 API 解密后只输出掩码而非明文密钥。
func TestAccountResponseMasksEncryptedCredentials(t *testing.T) {
	cipher, err := security.NewSecretCipher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cipher.Encrypt("telegram-super-secret")
	if err != nil {
		t.Fatal(err)
	}
	svcCtx := &svc.ServiceContext{SecretCipher: cipher}
	got, err := accountResponse(svcCtx, &models.ChannelAccount{EncryptedCredentials: models.JSONMap{"bot_token": encrypted}})
	if err != nil {
		t.Fatal(err)
	}
	if got.CredentialMasked["bot_token"] == "telegram-super-secret" || got.CredentialMasked["bot_token"] == "" {
		t.Fatalf("credential was not masked: %q", got.CredentialMasked["bot_token"])
	}
}

// TestProviderResponsePreservesCapabilities 验证控制面转换不会丢失能力声明和表单字段。
func TestProviderResponsePreservesCapabilities(t *testing.T) {
	got := providerResponse(corechannel.ProviderDescriptor{
		Name: corechannel.ProviderTelegram, DisplayName: "Telegram", MaxTextLength: 4096,
		CredentialFields: []corechannel.FieldDescriptor{{Key: "bot_token", Sensitive: true}},
		Capabilities:     corechannel.Capabilities{Typing: true, OutboundText: true},
	})
	if got.Name != "telegram" || !got.Capabilities.Typing || !got.Capabilities.OutboundText || len(got.CredentialFields) != 1 {
		t.Fatalf("unexpected provider response: %#v", got)
	}
}

// TestValidatePairingDecisionRejectsExpiredApproval 验证过期或已阻止的配对请求不能被批准。
func TestValidatePairingDecisionRejectsExpiredApproval(t *testing.T) {
	now := time.Now()
	expired := now.Add(-time.Minute)
	if err := validatePairingDecision(&models.ChannelIdentity{Status: models.ChannelIdentityStatusPending, PairingExpiresAt: &expired}, true, now); err == nil {
		t.Fatal("expected expired pairing to be rejected")
	}
	future := now.Add(time.Minute)
	if err := validatePairingDecision(&models.ChannelIdentity{Status: models.ChannelIdentityStatusBlocked, PairingExpiresAt: &future}, true, now); err == nil {
		t.Fatal("expected blocked pairing to be rejected")
	}
}
