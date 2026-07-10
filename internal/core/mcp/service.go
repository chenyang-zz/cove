package mcp

import (
	"context"
	"time"
)

// ToolClient 提供不复用 session 的 MCP 工具发现能力。
type ToolClient interface {
	ListTools(ctx context.Context, server ServerConfig) ([]ToolInfo, error)
}

// ToolSession 表示一次可复用的 MCP 服务连接。
//
// 同一个 session 的工具调用由 OpenedTools 串行化；Close 应允许重复调用。
type ToolSession interface {
	ListTools(ctx context.Context) ([]ToolInfo, error)
	CallTool(ctx context.Context, name string, input map[string]any) (*CallResult, error)
	Close() error
}

// SessionOpener 创建指定 MCP 服务的一次可复用连接。
type SessionOpener interface {
	OpenSession(ctx context.Context, server ServerConfig) (ToolSession, error)
}

// Options 配置 MCP 工具发现、会话建立和运行时缓存依赖。
type Options struct {
	Client        ToolClient
	SessionOpener SessionOpener
	Cache         ToolCache
	TTL           time.Duration
	Now           func() time.Time
}

// Service 统一管理 MCP 工具发现缓存和单轮可复用工具会话。
type Service struct {
	client        ToolClient
	sessionOpener SessionOpener
	cache         ToolCache
	ttl           time.Duration
	now           func() time.Time
}

// NewService 创建 MCP 服务；未提供依赖时使用 SDK client、内存缓存和五分钟 TTL。
func NewService(options Options) *Service {
	ttl := options.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	cache := options.Cache
	if cache == nil {
		cache = NewMemoryToolCache()
	}
	client := options.Client
	defaultClient := NewSDKToolClient(nil)
	if client == nil {
		client = defaultClient
	}
	sessionOpener := options.SessionOpener
	if sessionOpener == nil {
		if opener, ok := client.(SessionOpener); ok {
			sessionOpener = opener
		} else {
			sessionOpener = defaultClient
		}
	}
	return &Service{
		client:        client,
		sessionOpener: sessionOpener,
		cache:         cache,
		ttl:           ttl,
		now:           now,
	}
}

// BuildToolList 返回 MCP 工具列表；有效缓存直接命中，否则同步刷新远端。
func (s *Service) BuildToolList(ctx context.Context, server ServerConfig) ([]ToolInfo, error) {
	s = s.withDefaults()
	key := cacheKey(server)
	if entry, ok, err := s.cache.Get(ctx, key); err != nil {
		return nil, err
	} else if ok && s.cache.Valid(server, entry) {
		return cloneTools(entry.Tools), nil
	}

	return s.RefreshToolList(ctx, server)
}

// RefreshToolList 跳过 TTL 缓存并重新拉取远端 MCP 工具列表。
//
// 拉取成功后会更新运行时缓存；远端调用或缓存写入失败时返回对应错误。
func (s *Service) RefreshToolList(ctx context.Context, server ServerConfig) ([]ToolInfo, error) {
	s = s.withDefaults()
	tools, err := s.client.ListTools(ctx, server)
	if err != nil {
		return nil, err
	}
	tools = cloneTools(tools)
	if err := s.cache.Set(ctx, cacheKey(server), s.newEntry(server, tools)); err != nil {
		return nil, err
	}
	return tools, nil
}

// OpenTools 返回一组可在本轮调用的 MCP 工具及其连接 lease。
//
// 有效 TTL 缓存会直接提供工具元数据，并把 session 建立延迟到首次 CallTool；缓存
// 失效时会建立 session、刷新工具列表并复用该连接。调用方必须调用 Close。
func (s *Service) OpenTools(ctx context.Context, server ServerConfig) (*OpenedTools, error) {
	s = s.withDefaults()
	entry, ok, err := s.cache.Get(ctx, cacheKey(server))
	if err != nil {
		return nil, err
	}
	if ok && s.cache.Valid(server, entry) {
		return newOpenedTools(server, entry.Tools, s.sessionOpener, nil), nil
	}

	session, err := s.sessionOpener.OpenSession(ctx, server)
	if err != nil {
		return nil, err
	}
	tools, err := session.ListTools(ctx)
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	tools = cloneTools(tools)
	if err := s.cache.Set(ctx, cacheKey(server), s.newEntry(server, tools)); err != nil {
		_ = session.Close()
		return nil, err
	}
	return newOpenedTools(server, tools, s.sessionOpener, session), nil
}

// CacheStatus 返回指定 server 当前运行时缓存是否有效及其过期时间。
func (s *Service) CacheStatus(ctx context.Context, server ServerConfig) (CacheStatus, error) {
	s = s.withDefaults()
	key := cacheKey(server)
	entry, ok, err := s.cache.Get(ctx, key)
	if err != nil {
		return CacheStatus{}, err
	}
	if ok && s.cache.Valid(server, entry) {
		return CacheStatus{Valid: true, Source: "runtime", Fingerprint: entry.Fingerprint, ExpiresAt: entry.ExpiresAt}, nil
	}
	return CacheStatus{Valid: false, Fingerprint: Fingerprint(server)}, nil
}

func (s *Service) withDefaults() *Service {
	if s != nil {
		if s.ttl <= 0 {
			s.ttl = DefaultTTL
		}
		if s.now == nil {
			s.now = time.Now
		}
		if s.cache == nil {
			s.cache = NewMemoryToolCache()
		}
		if s.client == nil {
			client := NewSDKToolClient(nil)
			s.client = client
			if s.sessionOpener == nil {
				s.sessionOpener = client
			}
		}
		if s.sessionOpener == nil {
			if opener, ok := s.client.(SessionOpener); ok {
				s.sessionOpener = opener
			} else {
				s.sessionOpener = NewSDKToolClient(nil)
			}
		}
		return s
	}
	return NewService(Options{})
}

func (s *Service) newEntry(server ServerConfig, tools []ToolInfo) CacheEntry {
	return CacheEntry{
		Fingerprint: Fingerprint(server),
		ExpiresAt:   s.now().Add(s.ttl),
		Tools:       cloneTools(tools),
	}
}

func cacheKey(server ServerConfig) string {
	return server.ID.String()
}
