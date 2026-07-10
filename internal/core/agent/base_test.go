package agent

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
)

type testDecision struct {
	Input coretool.Input
}

type testStep struct {
	Input coretool.Input
}

type recordingHooks struct {
	events []string
}

func (h *recordingHooks) BeforeRun(ctx context.Context, state State[testDecision, testStep]) error {
	h.events = append(h.events, "before_run")
	return nil
}

func (h *recordingHooks) AfterRun(ctx context.Context, result Result[testStep], runErr error) error {
	h.events = append(h.events, "after_run")
	return nil
}

func (h *recordingHooks) BeforeTransition(ctx context.Context, state State[testDecision, testStep], transition Transition) error {
	h.events = append(h.events, "before_transition:"+string(transition.From)+"->"+string(transition.To))
	return nil
}

func (h *recordingHooks) AfterTransition(ctx context.Context, state State[testDecision, testStep], transition Transition) error {
	h.events = append(h.events, "after_transition:"+string(transition.From)+"->"+string(transition.To))
	return nil
}

func (h *recordingHooks) BeforeModel(ctx context.Context, state State[testDecision, testStep], messages []*llm.Message) error {
	h.events = append(h.events, "before_model")
	return nil
}

func (h *recordingHooks) OnToken(ctx context.Context, state State[testDecision, testStep], text string) error {
	h.events = append(h.events, "on_token")
	return nil
}

func (h *recordingHooks) AfterModel(ctx context.Context, state State[testDecision, testStep], output string, modelErr error) error {
	h.events = append(h.events, "after_model")
	return nil
}

func (h *recordingHooks) AfterParse(ctx context.Context, state State[testDecision, testStep], decision testDecision, parseErr error) error {
	h.events = append(h.events, "after_parse")
	return nil
}

func (h *recordingHooks) BeforeTool(ctx context.Context, state State[testDecision, testStep], call ToolCall) error {
	h.events = append(h.events, "before_tool")
	return nil
}

func (h *recordingHooks) AfterTool(ctx context.Context, state State[testDecision, testStep], call ToolCall, output coretool.Output, toolErr error) error {
	h.events = append(h.events, "after_tool")
	return nil
}

func (h *recordingHooks) OnStep(ctx context.Context, state State[testDecision, testStep], step testStep) error {
	h.events = append(h.events, "on_step")
	return nil
}

func (h *recordingHooks) OnError(ctx context.Context, state State[testDecision, testStep], err error) error {
	h.events = append(h.events, "on_error")
	return nil
}

// 验证点：NewBase 在 registry 为 nil 时会创建空注册表和可用的工具 runner。
func TestNewBaseCreatesEmptyRegistryAndRunner(t *testing.T) {
	base := NewBase[testDecision, testStep](nil, nil)

	state := base.NewState(Input{Query: "hello"})
	if len(state.Tools) != 0 {
		t.Fatalf("NewBase(nil registry).NewState().Tools len = %d, want 0", len(state.Tools))
	}
	output, err := base.InvokeTool(context.Background(), "missing", nil)
	if err != nil {
		t.Fatalf("Base.InvokeTool(missing) error = %v, want nil default error output", err)
	}
	if output.Text == "" {
		t.Fatalf("Base.InvokeTool(missing).Text = empty, want error observation text")
	}
}

// 验证点：Transition 会触发 before/after hooks，并更新状态阶段。
func TestBaseTransitionRunsHooksAndUpdatesPhase(t *testing.T) {
	hooks := &recordingHooks{}
	base := NewBase[testDecision, testStep](nil, nil, WithHooks[testDecision, testStep](hooks))
	state := State[testDecision, testStep]{Phase: PhaseStart, Iteration: 2}

	if err := base.Transition(context.Background(), &state, PhaseModel, "call model"); err != nil {
		t.Fatalf("Base.Transition() error = %v, want nil", err)
	}
	if state.Phase != PhaseModel {
		t.Fatalf("Base.Transition() phase = %q, want %q", state.Phase, PhaseModel)
	}
	want := []string{"before_transition:start->model", "after_transition:start->model"}
	if !reflect.DeepEqual(hooks.events, want) {
		t.Fatalf("transition hook events = %#v, want %#v", hooks.events, want)
	}
}

// 验证点：FinishWithError 会设置停止原因，并触发 OnError 和 AfterRun。
func TestBaseFinishWithErrorSetsStopReasonAndRunsHooks(t *testing.T) {
	hooks := &recordingHooks{}
	base := NewBase[testDecision, testStep](nil, nil, WithHooks[testDecision, testStep](hooks))
	state := State[testDecision, testStep]{Phase: PhaseModel}
	result := Result[testStep]{Iterations: 3}

	got, err := base.FinishWithError(context.Background(), state, result, ErrMaxIterations)
	if !errors.Is(err, ErrMaxIterations) {
		t.Fatalf("Base.FinishWithError() error = %v, want ErrMaxIterations", err)
	}
	if got.StoppedBy != StopMaxIterations {
		t.Fatalf("Base.FinishWithError().StoppedBy = %q, want %q", got.StoppedBy, StopMaxIterations)
	}
	if !reflect.DeepEqual(hooks.events, []string{"before_transition:model->error", "after_transition:model->error", "on_error", "after_run"}) {
		t.Fatalf("finish error hook events = %#v, want transition/on_error/after_run", hooks.events)
	}
}

// 验证点：CloneState 和 CloneResult 不应泄漏 messages、tools、decision、steps 中的可变引用。
func TestBaseCloneStateAndResultCopiesMutableFields(t *testing.T) {
	base := NewBase[testDecision, testStep](nil, nil)
	base.SetCloneFuncs(func(decision testDecision) testDecision {
		decision.Input = cloneInput(decision.Input)
		return decision
	}, func(step testStep) testStep {
		step.Input = cloneInput(step.Input)
		return step
	})
	strict := true
	state := State[testDecision, testStep]{
		Input: Input{Messages: []*llm.Message{{Role: llm.UserRole, Content: "original"}}},
		Tools: []coretool.Descriptor{{
			Name: "tool",
			Schema: coretool.Schema{Strict: &strict, Parameters: coretool.ParametersSchema{
				Properties: map[string]coretool.PropertySchema{"query": {"type": "string"}},
			}},
		}},
		LastDecision: testDecision{Input: coretool.Input{"query": "old"}},
		Steps:        []testStep{{Input: coretool.Input{"step": "old"}}},
	}

	clonedState := base.CloneState(state)
	clonedResult := base.CloneResult(Result[testStep]{Steps: state.Steps})
	state.Input.Messages[0].Content = "changed"
	state.Tools[0].Schema.Parameters.Properties["query"]["type"] = "number"
	state.LastDecision.Input["query"] = "changed"
	state.Steps[0].Input["step"] = "changed"

	if clonedState.Input.Messages[0].Content != "original" {
		t.Fatalf("CloneState().Input.Messages[0].Content = %q, want original", clonedState.Input.Messages[0].Content)
	}
	if clonedState.Tools[0].Schema.Parameters.Properties["query"]["type"] != "string" {
		t.Fatalf("CloneState().Tools schema type = %#v, want string", clonedState.Tools[0].Schema.Parameters.Properties["query"]["type"])
	}
	if clonedState.LastDecision.Input["query"] != "old" {
		t.Fatalf("CloneState().LastDecision.Input[query] = %#v, want old", clonedState.LastDecision.Input["query"])
	}
	if clonedState.Steps[0].Input["step"] != "old" {
		t.Fatalf("CloneState().Steps[0].Input[step] = %#v, want old", clonedState.Steps[0].Input["step"])
	}
	if clonedResult.Steps[0].Input["step"] != "old" {
		t.Fatalf("CloneResult().Steps[0].Input[step] = %#v, want old", clonedResult.Steps[0].Input["step"])
	}
}

func cloneInput(input coretool.Input) coretool.Input {
	out := make(coretool.Input, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
