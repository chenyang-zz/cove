package agent

import (
	"context"

	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
)

// NoopHooks 是不执行任何副作用的默认 hooks。
type NoopHooks[D any, S any] struct{}

// BeforeRun 在运行开始前调用。
func (NoopHooks[D, S]) BeforeRun(ctx context.Context, state State[D, S]) error { return nil }

// AfterRun 在运行结束后调用。
func (NoopHooks[D, S]) AfterRun(ctx context.Context, result Result[S], runErr error) error {
	return nil
}

// BeforeTransition 在状态迁移前调用。
func (NoopHooks[D, S]) BeforeTransition(ctx context.Context, state State[D, S], transition Transition) error {
	return nil
}

// AfterTransition 在状态迁移后调用。
func (NoopHooks[D, S]) AfterTransition(ctx context.Context, state State[D, S], transition Transition) error {
	return nil
}

// BeforeModel 在模型调用前调用。
func (NoopHooks[D, S]) BeforeModel(ctx context.Context, state State[D, S], messages []*llm.Message) error {
	return nil
}

// OnToken 在模型返回可展示文本增量时调用。
func (NoopHooks[D, S]) OnToken(ctx context.Context, state State[D, S], text string) error { return nil }

// AfterModel 在模型调用后调用。
func (NoopHooks[D, S]) AfterModel(ctx context.Context, state State[D, S], output string, modelErr error) error {
	return nil
}

// AfterParse 在模型输出解析后调用。
func (NoopHooks[D, S]) AfterParse(ctx context.Context, state State[D, S], decision D, parseErr error) error {
	return nil
}

// BeforeTool 在工具调用前调用。
func (NoopHooks[D, S]) BeforeTool(ctx context.Context, state State[D, S], call ToolCall) error {
	return nil
}

// AfterTool 在工具调用后调用。
func (NoopHooks[D, S]) AfterTool(ctx context.Context, state State[D, S], call ToolCall, output coretool.Output, toolErr error) error {
	return nil
}

// OnStep 在单次 step 记录后调用。
func (NoopHooks[D, S]) OnStep(ctx context.Context, state State[D, S], step S) error { return nil }

// OnError 在运行错误发生后调用。
func (NoopHooks[D, S]) OnError(ctx context.Context, state State[D, S], err error) error {
	return nil
}
