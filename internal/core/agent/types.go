package agent

import (
	"context"
	"errors"

	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
)

// ErrMaxIterations 表示 Agent 在达到最大迭代次数后仍未得到最终答案。
var ErrMaxIterations = errors.New("max iterations reached")

// Input 表示一次 Agent 运行的输入。
//
// Query 用于最常见的单轮用户问题。Messages 用于调用方传入已有上下文；
// 两者同时存在时，具体实现可同时保留 Messages 并追加 Query。
type Input struct {
	Query    string
	Messages []*llm.Message
}

// Result 表示一次 Agent 运行的结果。
type Result[S any] struct {
	Answer     string
	Steps      []S
	Iterations int
	StoppedBy  StopReason
}

// StopReason 表示 Agent 停止运行的原因。
type StopReason string

const (
	// StopFinalAnswer 表示 Agent 得到了最终答案。
	StopFinalAnswer StopReason = "final_answer"
	// StopMaxIterations 表示 Agent 达到最大迭代次数。
	StopMaxIterations StopReason = "max_iterations"
	// StopError 表示 Agent 因错误停止。
	StopError StopReason = "error"
)

// Phase 表示 Agent 内部状态机阶段。
type Phase string

const (
	// PhaseStart 表示运行刚初始化。
	PhaseStart Phase = "start"
	// PhaseBuildPrompt 表示正在构造模型输入。
	PhaseBuildPrompt Phase = "build_prompt"
	// PhaseModel 表示正在调用模型。
	PhaseModel Phase = "model"
	// PhaseFallback 表示模型原生工具调用路径退回文本协议。
	PhaseFallback Phase = "fallback"
	// PhaseParse 表示正在解析模型输出。
	PhaseParse Phase = "parse"
	// PhaseTool 表示正在调用工具。
	PhaseTool Phase = "tool"
	// PhaseObserve 表示正在记录工具观察结果。
	PhaseObserve Phase = "observe"
	// PhaseFinish 表示运行正常结束。
	PhaseFinish Phase = "finish"
	// PhaseError 表示运行因错误结束。
	PhaseError Phase = "error"
)

// Transition 表示一次状态阶段迁移。
type Transition struct {
	From      Phase
	To        Phase
	Iteration int
	Reason    string
}

// State 表示 Agent 当前运行状态。
//
// State 会传给 prompt builder 和 hooks。调用方应把它视为快照，不应依赖修改它来影响
// Agent 内部状态。
type State[D any, S any] struct {
	Input        Input
	Tools        []coretool.Descriptor
	Steps        []S
	Iteration    int
	Phase        Phase
	LastDecision D
	LastError    error
	SystemPrompt string
}

// ToolCall 表示 Agent 准备执行的一次工具调用。
type ToolCall struct {
	Name  string
	Input coretool.Input
}

// Hooks 定义 Agent 关键生命周期和状态迁移钩子。
type Hooks[D any, S any] interface {
	BeforeRun(ctx context.Context, state State[D, S]) error
	AfterRun(ctx context.Context, result Result[S], runErr error) error
	BeforeTransition(ctx context.Context, state State[D, S], transition Transition) error
	AfterTransition(ctx context.Context, state State[D, S], transition Transition) error
	BeforeModel(ctx context.Context, state State[D, S], messages []*llm.Message) error
	OnToken(ctx context.Context, state State[D, S], text string) error
	AfterModel(ctx context.Context, state State[D, S], output string, modelErr error) error
	AfterParse(ctx context.Context, state State[D, S], decision D, parseErr error) error
	BeforeTool(ctx context.Context, state State[D, S], call ToolCall) error
	AfterTool(ctx context.Context, state State[D, S], call ToolCall, output coretool.Output, toolErr error) error
	OnStep(ctx context.Context, state State[D, S], step S) error
	OnError(ctx context.Context, state State[D, S], err error) error
}
