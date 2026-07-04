package tool

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// 验证点：函数工具应返回注册时的描述，并把输入交给调用函数处理。
func TestFuncToolDescribesAndInvokes(t *testing.T) {
	ctx := context.Background()
	desc := Descriptor{
		Name:        "search",
		Description: "Search documents.",
		InputSchema: map[string]any{
			"type": "object",
		},
	}
	ft := NewFuncTool(desc, func(ctx context.Context, input Input) (Output, error) {
		return Output{
			Text: "query=" + input["query"].(string),
			Metadata: map[string]any{
				"count": 1,
			},
		}, nil
	})

	gotDesc, err := ft.Describe(ctx)
	if err != nil {
		t.Fatalf("FuncTool.Describe() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(gotDesc, desc) {
		t.Fatalf("FuncTool.Describe() = %#v, want %#v", gotDesc, desc)
	}

	got, err := ft.Invoke(ctx, Input{"query": "rag"})
	if err != nil {
		t.Fatalf("FuncTool.Invoke() error = %v, want nil", err)
	}
	if got.Text != "query=rag" {
		t.Fatalf("FuncTool.Invoke().Text = %q, want %q", got.Text, "query=rag")
	}
	if got.Metadata["count"] != 1 {
		t.Fatalf("FuncTool.Invoke().Metadata[count] = %#v, want 1", got.Metadata["count"])
	}
}

// 验证点：注册表应拒绝 nil 工具、空名称工具和重复名称工具，避免运行期歧义。
func TestRegistryRejectsInvalidTools(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	if err := registry.Register(ctx, nil); err == nil {
		t.Fatalf("Registry.Register(nil) error = nil, want error")
	}

	emptyName := NewFuncTool(Descriptor{Name: " "}, func(ctx context.Context, input Input) (Output, error) {
		return Output{}, nil
	})
	if err := registry.Register(ctx, emptyName); err == nil {
		t.Fatalf("Registry.Register(empty name) error = nil, want error")
	}

	first := NewFuncTool(Descriptor{Name: "lookup"}, func(ctx context.Context, input Input) (Output, error) {
		return Output{Text: "first"}, nil
	})
	if err := registry.Register(ctx, first); err != nil {
		t.Fatalf("Registry.Register(first) error = %v, want nil", err)
	}

	duplicate := NewFuncTool(Descriptor{Name: "lookup"}, func(ctx context.Context, input Input) (Output, error) {
		return Output{Text: "duplicate"}, nil
	})
	if err := registry.Register(ctx, duplicate); err == nil {
		t.Fatalf("Registry.Register(duplicate) error = nil, want error")
	}
}

// 验证点：工具清单应按名称稳定排序，并支持 enabled 过滤。
func TestRegistryListReturnsStableDescriptors(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()
	for _, name := range []string{"zeta", "alpha", "beta"} {
		toolName := name
		err := registry.Register(ctx, NewFuncTool(Descriptor{Name: toolName}, func(ctx context.Context, input Input) (Output, error) {
			return Output{Text: toolName}, nil
		}))
		if err != nil {
			t.Fatalf("Registry.Register(%q) error = %v, want nil", toolName, err)
		}
	}

	got := registry.List(nil)
	gotNames := descriptorNames(got)
	wantNames := []string{"alpha", "beta", "zeta"}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("Registry.List(nil) names = %#v, want %#v", gotNames, wantNames)
	}

	got = registry.List(map[string]bool{"beta": true, "missing": true})
	gotNames = descriptorNames(got)
	wantNames = []string{"beta"}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("Registry.List(enabled) names = %#v, want %#v", gotNames, wantNames)
	}
}

// 验证点：Runner 应按工具名查找并调用工具，nil input 应被规整为空输入。
func TestRunnerInvokeCallsRegisteredTool(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()
	err := registry.Register(ctx, NewFuncTool(Descriptor{Name: "echo"}, func(ctx context.Context, input Input) (Output, error) {
		if input == nil {
			t.Fatalf("FuncTool input = nil, want empty map")
		}
		return Output{Text: "ok"}, nil
	}))
	if err != nil {
		t.Fatalf("Registry.Register(echo) error = %v, want nil", err)
	}

	got, err := NewRunner(registry).Invoke(ctx, "echo", nil)
	if err != nil {
		t.Fatalf("Runner.Invoke(echo) error = %v, want nil", err)
	}
	if got.Text != "ok" {
		t.Fatalf("Runner.Invoke(echo).Text = %q, want %q", got.Text, "ok")
	}
}

// 验证点：默认策略应把未知工具错误转成可观察输出，方便模型后续修正。
func TestRunnerInvokeUnknownToolReturnsErrorOutputByDefault(t *testing.T) {
	got, err := NewRunner(NewRegistry()).Invoke(context.Background(), "missing", Input{})
	if err != nil {
		t.Fatalf("Runner.Invoke(missing) error = %v, want nil", err)
	}
	if got.Text == "" {
		t.Fatalf("Runner.Invoke(missing).Text = empty, want error text")
	}
	if got.Metadata["error"] == nil {
		t.Fatalf("Runner.Invoke(missing).Metadata[error] = nil, want error metadata")
	}
}

// 验证点：严格策略下工具执行错误应直接返回 error，不包装成输出。
func TestRunnerInvokeReturnsErrorWhenErrorAsOutputDisabled(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()
	invokeErr := errors.New("backend unavailable")
	err := registry.Register(ctx, NewFuncTool(Descriptor{Name: "fail"}, func(ctx context.Context, input Input) (Output, error) {
		return Output{}, invokeErr
	}))
	if err != nil {
		t.Fatalf("Registry.Register(fail) error = %v, want nil", err)
	}

	got, err := NewRunner(registry, WithErrorAsOutput(false)).Invoke(ctx, "fail", Input{})
	if !errors.Is(err, invokeErr) {
		t.Fatalf("Runner.Invoke(fail) error = %v, want %v", err, invokeErr)
	}
	if !reflect.DeepEqual(got, Output{}) {
		t.Fatalf("Runner.Invoke(fail) output = %#v, want zero Output", got)
	}
}

// 验证点：工具输出应保留结构化 part 和 metadata，便于后续扩展图片或文件结果。
func TestOutputCarriesPartsAndMetadata(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()
	err := registry.Register(ctx, NewFuncTool(Descriptor{Name: "image"}, func(ctx context.Context, input Input) (Output, error) {
		return Output{
			Text: "done",
			Parts: []Part{
				{Type: "image", Data: []byte{1, 2, 3}, MIME: "image/png"},
				{Type: "text", Text: "caption"},
			},
			Metadata: map[string]any{"source": "unit-test"},
		}, nil
	}))
	if err != nil {
		t.Fatalf("Registry.Register(image) error = %v, want nil", err)
	}

	got, err := NewRunner(registry).Invoke(ctx, "image", Input{})
	if err != nil {
		t.Fatalf("Runner.Invoke(image) error = %v, want nil", err)
	}
	if len(got.Parts) != 2 {
		t.Fatalf("Runner.Invoke(image) parts len = %d, want 2", len(got.Parts))
	}
	if got.Parts[0].MIME != "image/png" || !reflect.DeepEqual(got.Parts[0].Data, []byte{1, 2, 3}) {
		t.Fatalf("Runner.Invoke(image) first part = %#v, want png bytes", got.Parts[0])
	}
	if got.Metadata["source"] != "unit-test" {
		t.Fatalf("Runner.Invoke(image).Metadata[source] = %#v, want unit-test", got.Metadata["source"])
	}
}

func descriptorNames(descriptors []Descriptor) []string {
	names := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		names = append(names, descriptor.Name)
	}
	return names
}
