// Package gateway 实现消息网关控制面用例。
package gateway

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/infrastructure/security"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

const reloadChannel = "gateway:reload"

// Service 提供网关管理 API 的用户隔离操作。
type Service struct {
	svc *svc.ServiceContext
	log *slog.Logger
}

// NewService 创建网关控制面服务。
func NewService(svcCtx *svc.ServiceContext) *Service {
	return &Service{svc: svcCtx, log: xlog.Component("logic.gateway")}
}

// Providers 返回编译进 Cove 的 Provider 描述。
func (s *Service) Providers() []response.GatewayProviderResponse {
	if s == nil || s.svc == nil || s.svc.ChannelRegistry == nil {
		return nil
	}
	descriptors := s.svc.ChannelRegistry.Descriptors()
	out := make([]response.GatewayProviderResponse, 0, len(descriptors))
	for _, descriptor := range descriptors {
		out = append(out, providerResponse(descriptor))
	}
	return out
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

// CreateAccount 创建并加密渠道账号。
func (s *Service) CreateAccount(ctx context.Context, userID uuid.UUID, input *request.CreateChannelAccountRequest) (*response.ChannelAccountResponse, error) {
	provider, ok := s.svc.ChannelRegistry.Get(corechannel.ProviderName(input.Provider))
	if !ok {
		return nil, xerr.BadRequest("不支持的渠道 Provider")
	}
	if err := validateFields(provider.Descriptor().CredentialFields, input.Credentials, true); err != nil {
		return nil, err
	}
	if err := validateSettings(provider.Descriptor().SettingFields, input.Settings, true); err != nil {
		return nil, err
	}
	if input.DefaultAgentConfigID != nil {
		if _, err := s.svc.AgentConfigRepo.FindByID(ctx, userID, *input.DefaultAgentConfigID); err != nil {
			return nil, err
		}
	}
	encrypted, err := encryptCredentials(s.svc.SecretCipher, input.Credentials)
	if err != nil {
		return nil, err
	}
	publicID, err := randomPublicID()
	if err != nil {
		return nil, xerr.Internal("生成渠道公开 ID 失败", err)
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	row, err := s.svc.ChannelGatewayRepo.CreateAccount(ctx, userID, &models.ChannelAccount{
		Provider: input.Provider, Name: strings.TrimSpace(input.Name), PublicID: publicID,
		EncryptedCredentials: encrypted, Settings: cloneSettings(input.Settings),
		DefaultAgentConfigID: input.DefaultAgentConfigID, Enabled: enabled, Status: models.ChannelAccountStatusUnknown,
	})
	if err != nil {
		return nil, err
	}
	s.publishReload(ctx, row.ID)
	return s.accountResponse(row)
}

// ListAccounts 列出当前用户的账号。
func (s *Service) ListAccounts(ctx context.Context, userID uuid.UUID) ([]*response.ChannelAccountResponse, error) {
	rows, err := s.svc.ChannelGatewayRepo.ListAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ChannelAccountResponse, 0, len(rows))
	for _, row := range rows {
		item, mapErr := s.accountResponse(row)
		if mapErr != nil {
			return nil, mapErr
		}
		out = append(out, item)
	}
	return out, nil
}

// GetAccount 获取当前用户的渠道账号。
func (s *Service) GetAccount(ctx context.Context, userID, accountID uuid.UUID) (*response.ChannelAccountResponse, error) {
	row, err := s.svc.ChannelGatewayRepo.FindAccountByID(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	return s.accountResponse(row)
}

// UpdateAccount 局部更新账号并发布重载通知。
func (s *Service) UpdateAccount(ctx context.Context, userID, accountID uuid.UUID, input *request.UpdateChannelAccountRequest) (*response.ChannelAccountResponse, error) {
	current, err := s.svc.ChannelGatewayRepo.FindAccountByID(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	provider, ok := s.svc.ChannelRegistry.Get(corechannel.ProviderName(current.Provider))
	if !ok {
		return nil, xerr.BadRequest("渠道 Provider 不可用")
	}
	values := make(map[string]any)
	if input.Name != nil {
		values["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Credentials != nil {
		if err := validateFields(provider.Descriptor().CredentialFields, input.Credentials, false); err != nil {
			return nil, err
		}
		encrypted := cloneSettings(current.EncryptedCredentials)
		for key, value := range input.Credentials {
			ciphertext, encryptErr := s.svc.SecretCipher.Encrypt(value)
			if encryptErr != nil {
				return nil, xerr.Internal("渠道凭据加密失败", encryptErr)
			}
			encrypted[key] = ciphertext
		}
		values["encrypted_credentials"] = encrypted
	}
	if input.Settings != nil {
		if err := validateSettings(provider.Descriptor().SettingFields, input.Settings, true); err != nil {
			return nil, err
		}
		values["settings"] = cloneSettings(input.Settings)
	}
	if input.DefaultAgentConfigID != nil {
		if _, err := s.svc.AgentConfigRepo.FindByID(ctx, userID, *input.DefaultAgentConfigID); err != nil {
			return nil, err
		}
		values["default_agent_config_id"] = *input.DefaultAgentConfigID
	} else if input.ClearDefaultAgent {
		values["default_agent_config_id"] = nil
	}
	if input.Enabled != nil {
		values["enabled"] = *input.Enabled
		if !*input.Enabled {
			values["status"] = models.ChannelAccountStatusDisabled
		}
	}
	if len(values) == 0 {
		return nil, xerr.BadRequest("更新字段不能为空")
	}
	row, err := s.svc.ChannelGatewayRepo.UpdateAccount(ctx, userID, accountID, values)
	if err != nil {
		return nil, err
	}
	s.publishReload(ctx, accountID)
	return s.accountResponse(row)
}

// DeleteAccount 删除用户拥有的账号及其级联数据。
func (s *Service) DeleteAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	if err := s.svc.ChannelGatewayRepo.DeleteAccount(ctx, userID, accountID); err != nil {
		return err
	}
	s.publishReload(ctx, accountID)
	return nil
}

// TestAccount 使用 Provider 的可选 Tester 验证账号凭据。
func (s *Service) TestAccount(ctx context.Context, userID, accountID uuid.UUID) error {
	row, err := s.svc.ChannelGatewayRepo.FindAccountByID(ctx, userID, accountID)
	if err != nil {
		return err
	}
	provider, ok := s.svc.ChannelRegistry.Get(corechannel.ProviderName(row.Provider))
	if !ok {
		return xerr.BadRequest("渠道 Provider 不可用")
	}
	tester, ok := provider.(corechannel.Tester)
	if !ok {
		return nil
	}
	account, err := s.AccountConfig(row)
	if err != nil {
		return err
	}
	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := tester.TestAccount(testCtx, account); err != nil {
		_ = s.svc.ChannelGatewayRepo.UpdateAccountHealth(ctx, accountID, models.ChannelAccountStatusDegraded, "连接测试失败")
		return xerr.BadRequest("渠道账号连接测试失败")
	}
	return s.svc.ChannelGatewayRepo.UpdateAccountHealth(ctx, accountID, models.ChannelAccountStatusHealthy, "")
}

// AccountConfig 解密 Provider 运行所需的账号快照。
func (s *Service) AccountConfig(row *models.ChannelAccount) (corechannel.AccountConfig, error) {
	credentials := make(map[string]string, len(row.EncryptedCredentials))
	for key, raw := range row.EncryptedCredentials {
		ciphertext, ok := raw.(string)
		if !ok {
			return corechannel.AccountConfig{}, xerr.Internal("渠道凭据格式无效", nil)
		}
		plain, err := s.svc.SecretCipher.Decrypt(ciphertext)
		if err != nil {
			return corechannel.AccountConfig{}, xerr.Internal("渠道凭据解密失败", err)
		}
		credentials[key] = plain
	}
	return corechannel.AccountConfig{ID: row.ID.String(), PublicID: row.PublicID, Credentials: credentials, Settings: cloneSettings(row.Settings)}, nil
}

// ListPairings 列出账号的配对状态。
func (s *Service) ListPairings(ctx context.Context, userID, accountID uuid.UUID) ([]*response.ChannelPairingResponse, error) {
	if _, err := s.svc.ChannelGatewayRepo.FindAccountByID(ctx, userID, accountID); err != nil {
		return nil, err
	}
	rows, err := s.svc.ChannelGatewayRepo.ListPairings(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ChannelPairingResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, pairingResponse(row))
	}
	return out, nil
}

// DecidePairing 批准或阻止配对请求。
func (s *Service) DecidePairing(ctx context.Context, userID, identityID uuid.UUID, approve bool) (*response.ChannelPairingResponse, error) {
	row, err := s.svc.ChannelGatewayRepo.FindIdentityByID(ctx, userID, identityID)
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
	row, err = s.svc.ChannelGatewayRepo.UpdateIdentity(ctx, userID, identityID, values)
	if err != nil {
		return nil, err
	}
	return pairingResponse(row), nil
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

// ApprovePairing 批准当前用户拥有的私聊配对请求。
func (s *Service) ApprovePairing(ctx context.Context, userID, identityID uuid.UUID) (*response.ChannelPairingResponse, error) {
	return s.DecidePairing(ctx, userID, identityID, true)
}

// DenyPairing 拒绝并阻止当前用户拥有的私聊配对请求。
func (s *Service) DenyPairing(ctx context.Context, userID, identityID uuid.UUID) (*response.ChannelPairingResponse, error) {
	return s.DecidePairing(ctx, userID, identityID, false)
}

// ListBindings 查询用户的渠道绑定。
func (s *Service) ListBindings(ctx context.Context, userID uuid.UUID, accountID *uuid.UUID) ([]*response.ChannelBindingResponse, error) {
	if accountID != nil {
		if _, err := s.svc.ChannelGatewayRepo.FindAccountByID(ctx, userID, *accountID); err != nil {
			return nil, err
		}
	}
	rows, err := s.svc.ChannelGatewayRepo.ListBindings(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.ChannelBindingResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, bindingResponse(row))
	}
	return out, nil
}

// GetBinding 获取单个用户绑定。
func (s *Service) GetBinding(ctx context.Context, userID, bindingID uuid.UUID) (*response.ChannelBindingResponse, error) {
	row, err := s.svc.ChannelGatewayRepo.FindBindingByID(ctx, userID, bindingID)
	if err != nil {
		return nil, err
	}
	return bindingResponse(row), nil
}

// CreateBinding 创建显式白名单绑定。
func (s *Service) CreateBinding(ctx context.Context, userID uuid.UUID, input *request.CreateChannelBindingRequest) (*response.ChannelBindingResponse, error) {
	if _, err := s.svc.ChannelGatewayRepo.FindAccountByID(ctx, userID, input.AccountID); err != nil {
		return nil, err
	}
	if input.AgentConfigID != nil {
		if _, err := s.svc.AgentConfigRepo.FindByID(ctx, userID, *input.AgentConfigID); err != nil {
			return nil, err
		}
	}
	requireMention := input.ChatType == string(corechannel.ChatTypeGroup)
	toolPolicy := models.ChannelToolPolicyInherit
	if input.ChatType == string(corechannel.ChatTypeGroup) {
		toolPolicy = models.ChannelToolPolicySafe
	}
	enabled := true
	if input.RequireMention != nil {
		requireMention = *input.RequireMention
	}
	if input.ToolPolicy != nil {
		toolPolicy = *input.ToolPolicy
	}
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	routeKey := corechannel.StableRouteKey(input.AccountID.String(), input.ExternalChatID, input.ExternalThreadID)
	row, err := s.svc.ChannelGatewayRepo.CreateBinding(ctx, userID, &models.ChannelBinding{
		AccountID: input.AccountID, RouteKey: routeKey, ChatType: input.ChatType,
		ExternalChatID: input.ExternalChatID, ExternalThreadID: input.ExternalThreadID,
		AgentConfigID: input.AgentConfigID, RequireMention: requireMention, ToolPolicy: toolPolicy, Enabled: enabled,
	})
	if err != nil {
		return nil, err
	}
	return bindingResponse(row), nil
}

// UpdateBinding 更新绑定级 Agent 和工具策略。
func (s *Service) UpdateBinding(ctx context.Context, userID, bindingID uuid.UUID, input *request.UpdateChannelBindingRequest) (*response.ChannelBindingResponse, error) {
	values := make(map[string]any)
	if input.AgentConfigID != nil {
		if _, err := s.svc.AgentConfigRepo.FindByID(ctx, userID, *input.AgentConfigID); err != nil {
			return nil, err
		}
		values["agent_config_id"] = *input.AgentConfigID
	} else if input.ClearAgentConfig {
		values["agent_config_id"] = nil
	}
	if input.RequireMention != nil {
		values["require_mention"] = *input.RequireMention
	}
	if input.ToolPolicy != nil {
		values["tool_policy"] = *input.ToolPolicy
	}
	if input.Enabled != nil {
		values["enabled"] = *input.Enabled
	}
	if len(values) == 0 {
		return nil, xerr.BadRequest("更新字段不能为空")
	}
	row, err := s.svc.ChannelGatewayRepo.UpdateBinding(ctx, userID, bindingID, values)
	if err != nil {
		return nil, err
	}
	return bindingResponse(row), nil
}

// DeleteBinding 删除用户绑定。
func (s *Service) DeleteBinding(ctx context.Context, userID, bindingID uuid.UUID) error {
	return s.svc.ChannelGatewayRepo.DeleteBinding(ctx, userID, bindingID)
}

func (s *Service) accountResponse(row *models.ChannelAccount) (*response.ChannelAccountResponse, error) {
	plain, err := s.AccountConfig(row)
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

func (s *Service) publishReload(ctx context.Context, accountID uuid.UUID) {
	if s.svc.Redis != nil && s.svc.Redis.Raw() != nil {
		_ = s.svc.Redis.Raw().Publish(ctx, reloadChannel, accountID.String()).Err()
	}
}
