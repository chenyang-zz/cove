package channel

import (
	"context"
	"io"
)

// EventHandler 接收 Provider 已完成协议转换的规范化事件。
type EventHandler interface {
	HandleInbound(context.Context, InboundEvent) error
}

// EventHandlerFunc 允许普通函数作为 EventHandler 使用。
type EventHandlerFunc func(context.Context, InboundEvent) error

// HandleInbound 实现 EventHandler。
func (f EventHandlerFunc) HandleInbound(ctx context.Context, event InboundEvent) error {
	return f(ctx, event)
}

// Receiver 管理一个渠道账号的持续入站连接。
type Receiver interface {
	Receive(context.Context, AccountConfig, EventHandler) error
}

// Sender 负责文本发送和可选的输入状态通知。
type Sender interface {
	Send(context.Context, AccountConfig, OutboundMessage) (Receipt, error)
	SetTyping(context.Context, AccountConfig, Route, bool) error
}

// DownloadedMedia 是经过 Provider 官方 API 下载的受限媒体流。
type DownloadedMedia struct {
	Body     io.ReadCloser
	MIMEType string
	FileName string
	Size     int64
}

// MediaDownloader 是支持入站媒体的 Provider 可选实现。
type MediaDownloader interface {
	DownloadMedia(context.Context, AccountConfig, MediaReference) (*DownloadedMedia, error)
}

// Tester 是 Provider 可选的账号凭据连通性检查。
type Tester interface {
	TestAccount(context.Context, AccountConfig) error
}

// Provider 将静态描述、入站与出站能力组合成一个官方适配器。
type Provider interface {
	Descriptor() ProviderDescriptor
	Receiver
	Sender
}
