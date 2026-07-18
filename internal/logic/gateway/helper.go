package gateway

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/infrastructure/security"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

const reloadChannel = "gateway:reload"

func gatewayID(input *request.UriGatewayIDRequest) (uuid.UUID, error) {
	if input == nil {
		return uuid.Nil, xerr.BadRequest("网关资源参数不能为空")
	}
	id, err := uuid.Parse(strings.TrimSpace(input.ID))
	if err != nil {
		return uuid.Nil, xerr.BadRequest("ID 无效")
	}
	return id, nil
}

func providerResponse(descriptor corechannel.ProviderDescriptor) response.GatewayProviderResponse {
	fields := func(items []corechannel.FieldDescriptor) []response.GatewayFieldDescriptorResponse {
		out := make([]response.GatewayFieldDescriptorResponse, 0, len(items))
		for _, item := range items {
			out = append(out, response.GatewayFieldDescriptorResponse{
				Key: item.Key, Label: item.Label, Type: item.Type, Required: item.Required,
				Sensitive: item.Sensitive, Description: item.Description,
			})
		}
		return out
	}
	capabilities := descriptor.Capabilities
	return response.GatewayProviderResponse{
		Name: string(descriptor.Name), DisplayName: descriptor.DisplayName,
		Description: descriptor.Description, CredentialFields: fields(descriptor.CredentialFields),
		SettingFields: fields(descriptor.SettingFields), MaxTextLength: descriptor.MaxTextLength,
		Capabilities: response.GatewayCapabilitiesResponse{
			DirectMessages: capabilities.DirectMessages, GroupMessages: capabilities.GroupMessages,
			Threads: capabilities.Threads, Replies: capabilities.Replies, Mentions: capabilities.Mentions,
			Typing: capabilities.Typing, InboundImages: capabilities.InboundImages,
			InboundFiles: capabilities.InboundFiles, OutboundText: capabilities.OutboundText,
		},
	}
}

func accountConfig(svcCtx *svc.ServiceContext, row *models.ChannelAccount) (corechannel.AccountConfig, error) {
	if svcCtx == nil || svcCtx.SecretCipher == nil {
		return corechannel.AccountConfig{}, xerr.Internal("渠道凭据加密器未初始化", nil)
	}
	if row == nil {
		return corechannel.AccountConfig{}, xerr.BadRequest("渠道账号不能为空")
	}
	credentials := make(map[string]string, len(row.EncryptedCredentials))
	for key, raw := range row.EncryptedCredentials {
		ciphertext, ok := raw.(string)
		if !ok {
			return corechannel.AccountConfig{}, xerr.Internal("渠道凭据格式无效", nil)
		}
		plain, err := svcCtx.SecretCipher.Decrypt(ciphertext)
		if err != nil {
			return corechannel.AccountConfig{}, xerr.Internal("渠道凭据解密失败", err)
		}
		credentials[key] = plain
	}
	return corechannel.AccountConfig{ID: row.ID.String(), PublicID: row.PublicID, Credentials: credentials, Settings: cloneSettings(row.Settings)}, nil
}

func accountResponse(svcCtx *svc.ServiceContext, row *models.ChannelAccount) (*response.ChannelAccountResponse, error) {
	plain, err := accountConfig(svcCtx, row)
	if err != nil {
		return nil, err
	}
	masked := make(map[string]string, len(plain.Credentials))
	for key, value := range plain.Credentials {
		masked[key] = security.MaskSecret(value)
	}
	return &response.ChannelAccountResponse{
		ID: row.ID, Provider: row.Provider, Name: row.Name, PublicID: row.PublicID,
		CredentialMasked: masked, Settings: row.Settings, DefaultAgentConfigID: row.DefaultAgentConfigID,
		Enabled: row.Enabled, Status: row.Status, LastError: row.LastError, LastSeenAt: row.LastSeenAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func validateFields(fields []corechannel.FieldDescriptor, credentials map[string]string, requireAll bool) error {
	allowed := make(map[string]corechannel.FieldDescriptor, len(fields))
	for _, field := range fields {
		allowed[field.Key] = field
		if requireAll && field.Required && strings.TrimSpace(credentials[field.Key]) == "" {
			return xerr.BadRequest(fmt.Sprintf("缺少渠道凭据 %s", field.Key))
		}
	}
	for key, value := range credentials {
		if _, ok := allowed[key]; !ok {
			return xerr.BadRequest(fmt.Sprintf("未知渠道凭据 %s", key))
		}
		if strings.TrimSpace(value) == "" {
			return xerr.BadRequest(fmt.Sprintf("渠道凭据 %s 不能为空", key))
		}
	}
	return nil
}

func validateSettings(fields []corechannel.FieldDescriptor, settings map[string]any, requireAll bool) error {
	allowed := make(map[string]corechannel.FieldDescriptor, len(fields))
	for _, field := range fields {
		allowed[field.Key] = field
		if requireAll && field.Required {
			value, ok := settings[field.Key]
			if !ok || settingEmpty(value) {
				return xerr.BadRequest(fmt.Sprintf("缺少渠道设置 %s", field.Key))
			}
		}
	}
	for key, value := range settings {
		field, ok := allowed[key]
		if !ok {
			return xerr.BadRequest(fmt.Sprintf("未知渠道设置 %s", key))
		}
		if err := validateSettingValue(field, value); err != nil {
			return err
		}
	}
	return nil
}

func validateSettingValue(field corechannel.FieldDescriptor, value any) error {
	switch field.Type {
	case "text", "password", "url":
		text, ok := value.(string)
		if !ok || (field.Required && strings.TrimSpace(text) == "") {
			return xerr.BadRequest(fmt.Sprintf("渠道设置 %s 格式无效", field.Key))
		}
		if field.Type == "url" && strings.TrimSpace(text) != "" {
			parsed, err := url.ParseRequestURI(text)
			if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return xerr.BadRequest(fmt.Sprintf("渠道设置 %s 必须是 HTTP(S) URL", field.Key))
			}
		}
	case "string_list":
		var values []string
		switch typed := value.(type) {
		case []string:
			values = typed
		case []any:
			for _, item := range typed {
				text, ok := item.(string)
				if !ok {
					return xerr.BadRequest(fmt.Sprintf("渠道设置 %s 格式无效", field.Key))
				}
				values = append(values, text)
			}
		default:
			return xerr.BadRequest(fmt.Sprintf("渠道设置 %s 格式无效", field.Key))
		}
		for _, item := range values {
			if strings.TrimSpace(item) == "" {
				return xerr.BadRequest(fmt.Sprintf("渠道设置 %s 不能包含空值", field.Key))
			}
		}
	}
	return nil
}

func settingEmpty(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []string:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	default:
		return false
	}
}

func encryptCredentials(cipher interface{ Encrypt(string) (string, error) }, values map[string]string) (models.JSONMap, error) {
	out := make(models.JSONMap, len(values))
	for key, value := range values {
		encrypted, err := cipher.Encrypt(value)
		if err != nil {
			return nil, xerr.Internal("渠道凭据加密失败", err)
		}
		out[key] = encrypted
	}
	return out, nil
}

func cloneSettings(input map[string]any) models.JSONMap {
	out := make(models.JSONMap, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func randomPublicID() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func pairingResponse(row *models.ChannelIdentity) *response.ChannelPairingResponse {
	return &response.ChannelPairingResponse{
		ID: row.ID, AccountID: row.AccountID, ExternalUserID: row.ExternalUserID,
		ExternalChatID: row.ExternalChatID, DisplayName: row.DisplayName, Status: row.Status,
		PairingCodeMasked: row.PairingCodeMasked, PairingExpiresAt: row.PairingExpiresAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func bindingResponse(row *models.ChannelBinding) *response.ChannelBindingResponse {
	return &response.ChannelBindingResponse{
		ID: row.ID, AccountID: row.AccountID, RouteKey: row.RouteKey, ChatType: row.ChatType,
		ExternalChatID: row.ExternalChatID, ExternalThreadID: row.ExternalThreadID,
		ConversationID: row.ConversationID, AgentConfigID: row.AgentConfigID,
		RequireMention: row.RequireMention, ToolPolicy: row.ToolPolicy, Enabled: row.Enabled,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func validatePairingDecision(row *models.ChannelIdentity, approve bool, now time.Time) error {
	if row == nil {
		return xerr.BadRequest("配对请求无效")
	}
	if !approve {
		return nil
	}
	if row.Status == models.ChannelIdentityStatusAllowed {
		return nil
	}
	if row.Status != models.ChannelIdentityStatusPending {
		return xerr.BadRequest("配对请求状态不允许批准")
	}
	if row.PairingExpiresAt == nil || !row.PairingExpiresAt.After(now) {
		return xerr.BadRequest("配对请求已过期，请让外部用户重新发起")
	}
	return nil
}

func decidePairing(ctx context.Context, svcCtx *svc.ServiceContext, userID, identityID uuid.UUID, approve bool) (*response.ChannelPairingResponse, error) {
	row, err := svcCtx.ChannelGatewayRepo.FindIdentityByID(ctx, userID, identityID)
	if err != nil {
		return nil, err
	}
	if err := validatePairingDecision(row, approve, time.Now()); err != nil {
		return nil, err
	}
	values := map[string]any{"pairing_code_hash": "", "pairing_code_masked": "", "pairing_expires_at": nil}
	if approve {
		now := time.Now()
		values["status"] = models.ChannelIdentityStatusAllowed
		values["approved_at"] = &now
	} else {
		values["status"] = models.ChannelIdentityStatusBlocked
	}
	row, err = svcCtx.ChannelGatewayRepo.UpdateIdentity(ctx, userID, identityID, values)
	if err != nil {
		return nil, err
	}
	return pairingResponse(row), nil
}

func publishReload(ctx context.Context, svcCtx *svc.ServiceContext, accountID uuid.UUID) {
	if svcCtx != nil && svcCtx.Redis != nil && svcCtx.Redis.Raw() != nil {
		_ = svcCtx.Redis.Raw().Publish(ctx, reloadChannel, accountID.String()).Err()
	}
}
