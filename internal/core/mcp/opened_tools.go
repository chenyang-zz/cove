package mcp

import (
	"context"
	"fmt"
	"sync"
)

// OpenedTools 持有一组 MCP 工具元数据，并按需管理本轮可复用 session。
//
// CallTool 会串行调用同一个 session。Close 幂等；关闭后再次调用会返回错误。
type OpenedTools struct {
	mu      sync.Mutex
	server  ServerConfig
	tools   []ToolInfo
	opener  SessionOpener
	session ToolSession
	closed  bool
}

func newOpenedTools(server ServerConfig, tools []ToolInfo, opener SessionOpener, session ToolSession) *OpenedTools {
	return &OpenedTools{
		server:  server,
		tools:   cloneTools(tools),
		opener:  opener,
		session: session,
	}
}

// Tools 返回当前 lease 的独立工具元数据副本。
func (o *OpenedTools) Tools() []ToolInfo {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return cloneTools(o.tools)
}

// CallTool 调用远端 MCP 工具，并在缓存命中场景首次调用时延迟建立 session。
func (o *OpenedTools) CallTool(ctx context.Context, name string, input map[string]any) (*CallResult, error) {
	if o == nil {
		return nil, fmt.Errorf("opened tools is nil")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return nil, fmt.Errorf("opened tools is closed")
	}
	if o.session == nil {
		if o.opener == nil {
			return nil, fmt.Errorf("mcp session opener is nil")
		}
		session, err := o.opener.OpenSession(ctx, o.server)
		if err != nil {
			return nil, err
		}
		o.session = session
	}
	return o.session.CallTool(ctx, name, input)
}

// Close 关闭已建立的 session；尚未建立 session 时只标记 lease 已关闭。
func (o *OpenedTools) Close() error {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return nil
	}
	o.closed = true
	if o.session == nil {
		return nil
	}
	return o.session.Close()
}
