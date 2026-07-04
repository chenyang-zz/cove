package tool

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type registryEntry struct {
	descriptor Descriptor
	tool       Tool
}

// Registry 保存一组可调用工具。
//
// Registry 可以被多个 goroutine 并发读取和注册。同名工具不允许重复注册，
// 这样模型看到的工具名和实际调用目标保持一一对应。
type Registry struct {
	mu      sync.RWMutex
	entries map[string]registryEntry
}

// NewRegistry 创建空工具注册表。
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]registryEntry),
	}
}

// Register 注册一个工具。
//
// Register 会调用 tool.Describe(ctx) 获取工具名并校验唯一性。nil 工具、空名称、
// 描述失败和重复名称都会返回错误。
func (r *Registry) Register(ctx context.Context, tool Tool) error {
	if r == nil {
		return errors.New("registry is nil")
	}
	if tool == nil {
		return errors.New("tool is nil")
	}

	descriptor, err := tool.Describe(ctx)
	if err != nil {
		return fmt.Errorf("describe tool: %w", err)
	}
	descriptor.Name = strings.TrimSpace(descriptor.Name)
	if descriptor.Name == "" {
		return errors.New("tool name is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[descriptor.Name]; ok {
		return fmt.Errorf("tool %q already registered", descriptor.Name)
	}
	r.entries[descriptor.Name] = registryEntry{
		descriptor: cloneDescriptor(descriptor),
		tool:       tool,
	}
	return nil
}

// Lookup 按名称查找工具。
//
// 第二个返回值 reports whether 工具存在。name 会先做 strings.TrimSpace 规整。
func (r *Registry) Lookup(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	name = strings.TrimSpace(name)
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[name]
	return entry.tool, ok
}

// List 返回工具描述清单。
//
// enabled 为 nil 时返回全部工具；enabled 非 nil 时只返回 map 中值为 true 的工具。
// 返回结果按工具名升序排列，便于测试、缓存和模型上下文保持稳定。
func (r *Registry) List(enabled map[string]bool) []Descriptor {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := r.enabledNamesLocked(enabled)
	descriptors := make([]Descriptor, 0, len(names))
	for _, name := range names {
		descriptors = append(descriptors, cloneDescriptor(r.entries[name].descriptor))
	}
	return descriptors
}

// Tools 返回可调用工具清单。
//
// enabled 的语义与 List 一致，返回结果同样按工具名升序排列。
func (r *Registry) Tools(enabled map[string]bool) []Tool {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := r.enabledNamesLocked(enabled)
	tools := make([]Tool, 0, len(names))
	for _, name := range names {
		tools = append(tools, r.entries[name].tool)
	}
	return tools
}

// enabledNamesLocked 返回按名称排序的工具名列表。
func (r *Registry) enabledNamesLocked(enabled map[string]bool) []string {
	// 先收集名称再排序，避免 Go map 迭代顺序影响模型上下文和测试结果。
	names := make([]string, 0, len(r.entries))
	for name := range r.entries {
		if enabled != nil && !enabled[name] {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
