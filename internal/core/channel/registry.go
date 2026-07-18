package channel

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Registry 保存编译进 Cove 的官方 Provider。
type Registry struct {
	mu        sync.RWMutex
	providers map[ProviderName]Provider
}

// NewRegistry 创建空注册表并应用可选的预注册 Provider。
func NewRegistry(options ...RegistryOption) (*Registry, error) {
	registry := &Registry{providers: make(map[ProviderName]Provider)}
	for _, option := range options {
		if option != nil {
			if err := option(registry); err != nil {
				return nil, err
			}
		}
	}
	return registry, nil
}

// Register 注册 Provider；同名 Provider 冲突时拒绝覆盖。
func (r *Registry) Register(provider Provider) error {
	if r == nil {
		return errors.New("channel registry is nil")
	}
	if provider == nil {
		return errors.New("channel provider is nil")
	}
	descriptor := provider.Descriptor()
	if err := descriptor.Validate(); err != nil {
		return fmt.Errorf("invalid provider descriptor: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[descriptor.Name]; exists {
		return fmt.Errorf("provider %q is already registered", descriptor.Name)
	}
	r.providers[descriptor.Name] = provider
	return nil
}

// Get 按稳定名称查找 Provider。
func (r *Registry) Get(name ProviderName) (Provider, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[name]
	return provider, ok
}

// Descriptors 返回按名称排序的能力描述，保证 API 输出稳定。
func (r *Registry) Descriptors() []ProviderDescriptor {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	items := make([]ProviderDescriptor, 0, len(r.providers))
	for _, provider := range r.providers {
		items = append(items, provider.Descriptor())
	}
	r.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}
