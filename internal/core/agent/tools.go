package agent

import (
	"context"
	"fmt"

	"github.com/boxify/api-go/internal/domain"
)

type Registry struct {
	builders map[string]func() domain.Tool
}

func NewRegistry() *Registry {
	return &Registry{builders: map[string]func() domain.Tool{}}
}

func (r *Registry) Register(key string, builder func() domain.Tool) {
	r.builders[key] = builder
}

func (r *Registry) Build(enabled map[string]bool) []domain.Tool {
	tools := make([]domain.Tool, 0, len(r.builders))
	for key, builder := range r.builders {
		if enabled != nil && !enabled[key] {
			continue
		}
		tools = append(tools, builder())
	}
	return tools
}

type EchoTool struct{}

func (EchoTool) Name() string {
	return "datetime"
}

func (EchoTool) Schema() domain.ToolSchema {
	return domain.ToolSchema{Name: "datetime", Description: "Return a deterministic placeholder tool result."}
}

func (EchoTool) Run(ctx context.Context, toolCtx domain.ToolContext, input domain.ToolInput) (domain.ToolResult, error) {
	query, _ := input["query"].(string)
	if query == "" {
		query = "now"
	}
	return domain.ToolResult{Text: fmt.Sprintf("tool:%s", query)}, nil
}
