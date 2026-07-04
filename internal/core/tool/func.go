package tool

import (
	"context"
	"errors"
)

// FuncTool 把普通 Go 函数包装成 Tool。
type FuncTool struct {
	descriptor Descriptor
	invoke     InvokeFunc
}

// NewFuncTool 使用 descriptor 和 invoke 创建函数工具。
//
// descriptor 会在返回时浅拷贝，避免调用方后续修改 map 字段影响工具元信息。
// invoke 为 nil 时工具仍可创建，但调用 Invoke 会返回错误，方便注册阶段统一校验名称。
func NewFuncTool(descriptor Descriptor, invoke InvokeFunc) *FuncTool {
	return &FuncTool{
		descriptor: cloneDescriptor(descriptor),
		invoke:     invoke,
	}
}

// Describe 返回函数工具的描述信息。
func (t *FuncTool) Describe(ctx context.Context) (Descriptor, error) {
	if t == nil {
		return Descriptor{}, errors.New("tool is nil")
	}
	return cloneDescriptor(t.descriptor), nil
}

// Invoke 执行函数工具。
func (t *FuncTool) Invoke(ctx context.Context, input Input) (Output, error) {
	if t == nil {
		return Output{}, errors.New("tool is nil")
	}
	if t.invoke == nil {
		return Output{}, errors.New("tool invoke function is nil")
	}
	if input == nil {
		input = Input{}
	}
	return t.invoke(ctx, input)
}
