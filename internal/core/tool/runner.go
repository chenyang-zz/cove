package tool

import (
	"context"
	"fmt"
	"strings"
)

// Runner 按工具名从 Registry 查找并调用工具。
type Runner struct {
	registry      *Registry
	errorAsOutput bool
}

// NewRunner 创建工具调用器。
//
// registry 为 nil 时会使用空注册表。默认错误策略是把错误转成 Output，
// 这样 agent 可以把错误作为观察结果继续修正参数或选择其他工具。
func NewRunner(registry *Registry, opts ...RunnerOption) *Runner {
	if registry == nil {
		registry = NewRegistry()
	}
	r := &Runner{
		registry:      registry,
		errorAsOutput: true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

// Invoke 调用指定名称的工具。
//
// name 会先做 strings.TrimSpace 规整。input 为 nil 时按空输入处理。默认配置下，
// 未知工具和工具执行错误会被转换成 Output；关闭 ErrorAsOutput 后则直接返回 error。
func (r *Runner) Invoke(ctx context.Context, name string, input Input) (Output, error) {
	if r == nil {
		return Output{}, fmt.Errorf("runner is nil")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return r.handleError(fmt.Errorf("tool name is empty"))
	}
	if input == nil {
		input = Input{}
	}

	tool, ok := r.registry.Lookup(name)
	if !ok {
		return r.handleError(fmt.Errorf("tool %q not found", name))
	}
	output, err := tool.Invoke(ctx, input)
	if err != nil {
		return r.handleError(err)
	}
	return output, nil
}

func (r *Runner) handleError(err error) (Output, error) {
	if err == nil {
		return Output{}, nil
	}
	if !r.errorAsOutput {
		return Output{}, err
	}
	return Output{
		Text: err.Error(),
		Metadata: map[string]any{
			"error": err.Error(),
		},
	}, nil
}
