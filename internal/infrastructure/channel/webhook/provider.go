// Package webhook 实现通用双向 HMAC Webhook Provider。
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	corechannel "github.com/boxify/api-go/internal/core/channel"
)

const defaultTimeout = 10 * time.Second

// Provider 将最终文本签名后回调到账号配置的 URL。
type Provider struct{ client *http.Client }

// New 创建通用 Webhook Provider。
func New(client *http.Client) *Provider {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &Provider{client: client}
}

// TestAccount 校验回调 URL 和签名密钥，不向第三方发送探测消息。
func (*Provider) TestAccount(_ context.Context, account corechannel.AccountConfig) error {
	callbackURL, _ := account.Settings["callback_url"].(string)
	parsed, err := url.ParseRequestURI(callbackURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return errors.New("webhook callback_url is invalid")
	}
	if strings.TrimSpace(account.Credentials["signing_secret"]) == "" {
		return errors.New("webhook signing_secret is required")
	}
	return nil
}

// Descriptor 返回配置表单和能力矩阵。
func (*Provider) Descriptor() corechannel.ProviderDescriptor {
	return corechannel.ProviderDescriptor{
		Name: corechannel.ProviderWebhook, DisplayName: "通用 Webhook",
		Description: "通过 HMAC 签名的入站请求与 HTTP 回调接入任意软件",
		CredentialFields: []corechannel.FieldDescriptor{
			{Key: "signing_secret", Label: "签名密钥", Type: "password", Required: true, Sensitive: true},
		},
		SettingFields: []corechannel.FieldDescriptor{
			{Key: "callback_url", Label: "回复回调地址", Type: "url", Required: true},
			{Key: "media_host_allowlist", Label: "媒体域名白名单", Type: "string_list"},
		},
		Capabilities: corechannel.Capabilities{
			DirectMessages: true, GroupMessages: true, Threads: true, Replies: true,
			Mentions: true, InboundImages: true, InboundFiles: true, OutboundText: true,
		},
		MaxTextLength: 16000,
	}
}

// Receive 等待进程退出；Webhook 入站由独立 HTTP 数据面驱动。
func (*Provider) Receive(ctx context.Context, _ corechannel.AccountConfig, _ corechannel.EventHandler) error {
	<-ctx.Done()
	return nil
}

// SetTyping 对不声明 typing 能力的 Webhook 静默降级。
func (*Provider) SetTyping(context.Context, corechannel.AccountConfig, corechannel.Route, bool) error {
	return nil
}

// Send 使用 delivery_id 作为下游幂等键并对完整 JSON 请求体签名。
func (p *Provider) Send(ctx context.Context, account corechannel.AccountConfig, message corechannel.OutboundMessage) (corechannel.Receipt, error) {
	callbackURL, _ := account.Settings["callback_url"].(string)
	secret := account.Credentials["signing_secret"]
	if strings.TrimSpace(callbackURL) == "" || secret == "" {
		return corechannel.Receipt{DeliveryID: message.DeliveryID, State: corechannel.DeliveryPermanent, ErrorCode: "invalid_config"}, errors.New("webhook callback config is incomplete")
	}
	payload := struct {
		DeliveryID string                      `json:"delivery_id"`
		Route      corechannel.Route           `json:"route"`
		Text       string                      `json:"text"`
		ReplyTo    *corechannel.ReplyReference `json:"reply_to,omitempty"`
	}{DeliveryID: message.DeliveryID, Route: message.Route, Text: message.Text, ReplyTo: message.ReplyTo}
	body, err := json.Marshal(payload)
	if err != nil {
		return corechannel.Receipt{DeliveryID: message.DeliveryID, State: corechannel.DeliveryPermanent}, err
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(body))
	if err != nil {
		return corechannel.Receipt{DeliveryID: message.DeliveryID, State: corechannel.DeliveryPermanent}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cove-Timestamp", timestamp)
	req.Header.Set("X-Cove-Signature", Sign(secret, timestamp, body))
	req.Header.Set("X-Cove-Delivery-ID", message.DeliveryID)
	resp, err := p.client.Do(req)
	if err != nil {
		return corechannel.Receipt{DeliveryID: message.DeliveryID, State: corechannel.DeliveryTemporary, ErrorMessage: err.Error()}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	receipt := corechannel.Receipt{DeliveryID: message.DeliveryID}
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		receipt.State = corechannel.DeliverySent
		return receipt, nil
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		receipt.State = corechannel.DeliveryTemporary
		receipt.ErrorCode = strconv.Itoa(resp.StatusCode)
		return receipt, fmt.Errorf("webhook callback returned %s", resp.Status)
	default:
		receipt.State = corechannel.DeliveryPermanent
		receipt.ErrorCode = strconv.Itoa(resp.StatusCode)
		return receipt, fmt.Errorf("webhook callback returned %s", resp.Status)
	}
}

// DownloadMedia 仅允许账号白名单域名，并在拨号前拒绝内网、回环和链路本地地址。
func (*Provider) DownloadMedia(ctx context.Context, account corechannel.AccountConfig, media corechannel.MediaReference) (*corechannel.DownloadedMedia, error) {
	allowlist := stringList(account.Settings["media_host_allowlist"])
	parsed, err := url.Parse(media.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return nil, errors.New("webhook media URL must be a valid https URL")
	}
	if !hostAllowed(parsed.Hostname(), allowlist) {
		return nil, errors.New("webhook media host is not allowlisted")
	}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(dialCtx context.Context, network, address string) (net.Conn, error) {
			host, port, splitErr := net.SplitHostPort(address)
			if splitErr != nil {
				return nil, splitErr
			}
			if !hostAllowed(host, allowlist) {
				return nil, errors.New("redirected media host is not allowlisted")
			}
			ips, lookupErr := net.DefaultResolver.LookupIPAddr(dialCtx, host)
			if lookupErr != nil {
				return nil, lookupErr
			}
			for _, resolved := range ips {
				if unsafeIP(resolved.IP) {
					return nil, errors.New("webhook media resolved to a private address")
				}
			}
			if len(ips) == 0 {
				return nil, errors.New("webhook media host has no address")
			}
			return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(dialCtx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 || !hostAllowed(req.URL.Hostname(), allowlist) || req.URL.Scheme != "https" {
				return errors.New("webhook media redirect is not allowed")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("webhook media returned %s", resp.Status)
	}
	size := resp.ContentLength
	if size <= 0 {
		size = media.Size
	}
	mimeType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if mimeType == "" {
		mimeType = media.MIMEType
	}
	return &corechannel.DownloadedMedia{Body: resp.Body, MIMEType: mimeType, FileName: media.FileName, Size: size}, nil
}

func stringList(value any) []string {
	switch values := value.(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, item := range values {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func hostAllowed(host string, allowlist []string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" || net.ParseIP(host) != nil {
		return false
	}
	for _, allowed := range allowlist {
		allowed = strings.ToLower(strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(allowed), "."), "."))
		if allowed != "" && (host == allowed || strings.HasSuffix(host, "."+allowed)) {
			return true
		}
	}
	return false
}

func unsafeIP(ip net.IP) bool {
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

// Sign 返回 timestamp + "." + body 的 HMAC-SHA256 十六进制摘要。
func Sign(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify 校验时间窗口与签名，抵御重放和时序侧信道。
func Verify(secret, timestamp, signature string, body []byte, now time.Time, window time.Duration) error {
	unix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid webhook timestamp")
	}
	delta := now.Sub(time.Unix(unix, 0))
	if delta < -window || delta > window {
		return errors.New("webhook timestamp outside allowed window")
	}
	expected, err := hex.DecodeString(Sign(secret, timestamp, body))
	if err != nil {
		return err
	}
	provided, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil || !hmac.Equal(expected, provided) {
		return errors.New("invalid webhook signature")
	}
	return nil
}
