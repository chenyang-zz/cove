package domain

import (
	"context"

	"github.com/google/uuid"
)

type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type ToolInput map[string]any

type ToolResult struct {
	Text      string         `json:"text"`
	Citations []Citation     `json:"citations,omitempty"`
	Stats     map[string]any `json:"stats,omitempty"`
}

type Citation struct {
	SourceID   string  `json:"source_id"`
	SourceType string  `json:"source_type"`
	Title      string  `json:"title"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score,omitempty"`
}

type ToolContext struct {
	UserID    uuid.UUID
	KBIDs     []string
	Citations *[]Citation
	Stats     map[string]map[string]any
}

type Tool interface {
	Name() string
	Schema() ToolSchema
	Run(ctx context.Context, toolCtx ToolContext, input ToolInput) (ToolResult, error)
}

type AgentEvent struct {
	Type      string         `json:"type"`
	Text      string         `json:"text,omitempty"`
	Tool      string         `json:"tool,omitempty"`
	Query     string         `json:"query,omitempty"`
	Status    string         `json:"status,omitempty"`
	Citations []Citation     `json:"citations,omitempty"`
	Stats     map[string]any `json:"stats,omitempty"`
	LatencyMS int64          `json:"latency_ms,omitempty"`
}
