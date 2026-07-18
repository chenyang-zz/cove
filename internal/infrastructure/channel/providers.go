// Package channel 组装编译进 Cove 的官方消息 Provider。
package channel

import (
	"net/http"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/infrastructure/channel/feishu"
	"github.com/boxify/api-go/internal/infrastructure/channel/telegram"
	"github.com/boxify/api-go/internal/infrastructure/channel/webhook"
)

// NewRegistry 返回包含 Telegram、飞书和通用 Webhook 的注册表。
// client 仅用于用户配置的 Webhook 回调；Telegram 保持独立的官方 API 超时边界。
func NewRegistry(client *http.Client) (*corechannel.Registry, error) {
	return corechannel.NewRegistry(corechannel.WithProviders(
		telegram.New(nil),
		feishu.New(),
		webhook.New(client),
	))
}
