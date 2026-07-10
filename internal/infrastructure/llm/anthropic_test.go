package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corellm "github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
	infra "github.com/boxify/api-go/internal/infrastructure/llm"
)

// 验证 Anthropic Invoke 会发送基础 Messages 请求并返回拼接文本。
func TestAnthropicClientInvokeSendsMessagesRequest(t *testing.T) {
	var authHeader string
	var path string
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("X-Api-Key")
		path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_123",
			"type":"message",
			"role":"assistant",
			"model":"claude-sonnet-4-5",
			"content":[
				{"type":"text","text":"hello"},
				{"type":"text","text":" world"}
			],
			"stop_reason":"end_turn",
			"stop_sequence":null,
			"usage":{"input_tokens":1,"output_tokens":2}
		}`))
	}))
	defer server.Close()

	client := infra.NewAnthropicLLMClient("sk-ant", "claude-sonnet-4-5", infra.WithAnthropicBaseURL(server.URL))
	got, err := client.Invoke(context.Background(), []*corellm.Message{
		corellm.SystemMessage("be precise"),
		corellm.UserMessage("ping"),
	}, corellm.WithTemperature(0.2), corellm.WithMaxTokens(128))
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if got != "hello world" {
		t.Fatalf("Invoke = %q, want hello world", got)
	}
	if path != "/v1/messages" {
		t.Fatalf("path = %q", path)
	}
	if authHeader != "sk-ant" {
		t.Fatalf("x-api-key = %q", authHeader)
	}
	if requestBody["model"] != "claude-sonnet-4-5" || requestBody["temperature"] != float64(0.2) || requestBody["max_tokens"] != float64(128) {
		t.Fatalf("request body = %#v", requestBody)
	}
	system, ok := requestBody["system"].([]any)
	if !ok || len(system) != 1 || system[0].(map[string]any)["text"] != "be precise" {
		t.Fatalf("system = %#v", requestBody["system"])
	}
	messages, ok := requestBody["messages"].([]any)
	if !ok || len(messages) != 1 || messages[0].(map[string]any)["role"] != "user" {
		t.Fatalf("messages = %#v", requestBody["messages"])
	}
}

// 验证 Anthropic InvokeResult 会发送工具参数，并映射文本、工具调用、停止原因和 token 用量。
func TestAnthropicClientInvokeResultMapsRichResponse(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_123",
			"type":"message",
			"role":"assistant",
			"model":"claude-sonnet-4-5",
			"content":[
				{"type":"text","text":"need tool"},
				{"type":"tool_use","id":"toolu_1","name":"search","input":{"query":"golang"}}
			],
			"stop_reason":"tool_use",
			"stop_sequence":null,
			"usage":{"input_tokens":3,"output_tokens":5,"cache_creation_input_tokens":2,"cache_read_input_tokens":1}
		}`))
	}))
	defer server.Close()

	client := infra.NewAnthropicLLMClient("sk-ant", "claude-sonnet-4-5", infra.WithAnthropicBaseURL(server.URL))
	strict := true
	result, err := client.InvokeResult(context.Background(),
		[]*corellm.Message{corellm.UserMessage("ping")},
		corellm.WithTopP(0.8),
		corellm.WithTools(coretool.Descriptor{
			Name:        "search",
			Description: "search docs",
			Schema: coretool.Schema{Strict: &strict, Parameters: coretool.ParametersSchema{
				Type: "object",
				Properties: map[string]coretool.PropertySchema{
					"query": {"type": "string"},
				},
				Required:             []string{"query"},
				AdditionalProperties: false,
			}},
		}),
		corellm.WithRequiredTool("search"),
	)
	if err != nil {
		t.Fatalf("InvokeResult error = %v, want nil", err)
	}
	if requestBody["top_p"] != float64(0.8) {
		t.Fatalf("request top_p = %#v, want 0.8; body=%#v", requestBody["top_p"], requestBody)
	}
	tools, ok := requestBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("request tools = %#v, want one tool", requestBody["tools"])
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "search" || tool["strict"] != true {
		t.Fatalf("request tool = %#v, want search strict tool", tool)
	}
	toolChoice := requestBody["tool_choice"].(map[string]any)
	if toolChoice["type"] != "tool" || toolChoice["name"] != "search" {
		t.Fatalf("request tool_choice = %#v, want search tool choice", toolChoice)
	}
	if result.Text != "need tool" || result.ID != "msg_123" || result.Model != "claude-sonnet-4-5" || result.Provider != "anthropic" || result.StopReason != "tool_use" {
		t.Fatalf("InvokeResult metadata = %#v, want mapped Anthropic fields", result)
	}
	if result.Usage.InputTokens != 3 || result.Usage.OutputTokens != 5 || result.Usage.CacheCreationInputTokens != 2 || result.Usage.CacheReadInputTokens != 1 || result.Usage.TotalTokens != 11 {
		t.Fatalf("InvokeResult usage = %#v, want total 11", result.Usage)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].ID != "toolu_1" || result.ToolCalls[0].Name != "search" || result.ToolCalls[0].Input["query"] != "golang" {
		t.Fatalf("InvokeResult tool calls = %#v, want parsed search call", result.ToolCalls)
	}
	if !strings.Contains(result.RawJSON, "msg_123") {
		t.Fatalf("InvokeResult RawJSON = %q, want original response", result.RawJSON)
	}
}

// 验证 Anthropic InvokeWithTools 会把工具定义、工具选择和历史工具结果映射到请求体。
func TestAnthropicClientInvokeWithToolsSendsToolHistory(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_456",
			"type":"message",
			"role":"assistant",
			"model":"claude-sonnet-4-5",
			"content":[{"type":"text","text":"done"}],
			"stop_reason":"end_turn",
			"usage":{"input_tokens":4,"output_tokens":2}
		}`))
	}))
	defer server.Close()

	client := infra.NewAnthropicLLMClient("sk-ant", "claude-sonnet-4-5", infra.WithAnthropicBaseURL(server.URL))
	result, err := client.(corellm.ToolCallingClient).InvokeWithTools(context.Background(),
		[]*corellm.Message{
			corellm.UserMessage("time?"),
			{
				Role: corellm.AssistantRole,
				ToolCalls: []corellm.LLMToolCall{
					{ID: "toolu_1", Name: "current_time", RawInput: `{"zone":"UTC"}`, Input: coretool.Input{"zone": "UTC"}},
				},
			},
			{Role: corellm.ToolRole, ToolCallID: "toolu_1", ToolName: "current_time", Content: "12:00"},
		},
		corellm.WithTools(coretool.Descriptor{
			Name:        "current_time",
			Description: "get current time",
			Schema: coretool.Schema{Parameters: coretool.ParametersSchema{
				Type:       "object",
				Properties: map[string]coretool.PropertySchema{"zone": {"type": "string"}},
				Required:   []string{"zone"},
			}},
		}),
		corellm.WithToolChoiceAuto(),
	)
	if err != nil {
		t.Fatalf("InvokeWithTools error = %v, want nil", err)
	}
	if result.Text != "done" || result.ID != "msg_456" {
		t.Fatalf("InvokeWithTools result = %#v, want done msg_456", result)
	}
	messages, ok := requestBody["messages"].([]any)
	if !ok || len(messages) != 3 {
		t.Fatalf("request messages = %#v, want user assistant tool_result", requestBody["messages"])
	}
	assistant := messages[1].(map[string]any)
	assistantContent := assistant["content"].([]any)
	toolUse := assistantContent[0].(map[string]any)
	if assistant["role"] != "assistant" || toolUse["type"] != "tool_use" || toolUse["id"] != "toolu_1" || toolUse["name"] != "current_time" {
		t.Fatalf("assistant message = %#v, want current_time tool_use", assistant)
	}
	toolResultMessage := messages[2].(map[string]any)
	toolResultContent := toolResultMessage["content"].([]any)
	toolResult := toolResultContent[0].(map[string]any)
	toolResultText := toolResult["content"].([]any)[0].(map[string]any)
	if toolResultMessage["role"] != "user" || toolResult["type"] != "tool_result" || toolResult["tool_use_id"] != "toolu_1" || toolResultText["text"] != "12:00" {
		t.Fatalf("tool result message = %#v, want toolu_1 result 12:00", toolResultMessage)
	}
	if _, ok := requestBody["tools"].([]any); !ok {
		t.Fatalf("request tools = %#v, want tools from WithTools", requestBody["tools"])
	}
	if requestBody["tool_choice"].(map[string]any)["type"] != "auto" {
		t.Fatalf("request tool_choice = %#v, want auto", requestBody["tool_choice"])
	}
}

// TestAnthropicClientPreservesCompleteToolJSONSchema 验证 Anthropic 工具参数会保留 MCP schema 的顶层扩展字段。
func TestAnthropicClientPreservesCompleteToolJSONSchema(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_schema","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"done"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	client := infra.NewAnthropicLLMClient("sk-ant", "claude-sonnet-4-5", infra.WithAnthropicBaseURL(server.URL))
	_, err := client.InvokeResult(context.Background(), []*corellm.Message{corellm.UserMessage("ping")}, corellm.WithTools(coretool.Descriptor{
		Name: "mcp_schema",
		Schema: coretool.Schema{Parameters: coretool.NewParametersSchema(map[string]any{
			"type":       "object",
			"properties": map[string]any{"query": map[string]any{"type": "string"}},
			"oneOf":      []any{map[string]any{"required": []any{"query"}}},
			"$defs":      map[string]any{"filter": map[string]any{"type": "object"}},
		})},
	}))
	if err != nil {
		t.Fatalf("InvokeResult error = %v, want nil", err)
	}
	tools := requestBody["tools"].([]any)
	inputSchema := tools[0].(map[string]any)["input_schema"].(map[string]any)
	if inputSchema["oneOf"] == nil || inputSchema["$defs"] == nil {
		t.Fatalf("Anthropic input_schema = %#v, want oneOf and $defs", inputSchema)
	}
}

// 验证 Anthropic Invoke 会使用默认最大 token。
func TestAnthropicClientInvokeUsesDefaultMaxTokens(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"msg_123",
			"type":"message",
			"role":"assistant",
			"model":"claude-sonnet-4-5",
			"content":[{"type":"text","text":"pong"}],
			"stop_reason":"end_turn",
			"stop_sequence":null,
			"usage":{"input_tokens":1,"output_tokens":1}
		}`))
	}))
	defer server.Close()

	client := infra.NewAnthropicLLMClient("sk-ant", "claude-sonnet-4-5", infra.WithAnthropicBaseURL(server.URL))
	if _, err := client.Invoke(context.Background(), []*corellm.Message{corellm.UserMessage("ping")}); err != nil {
		t.Fatalf("Invoke error = %v", err)
	}
	if requestBody["max_tokens"] != float64(1024) {
		t.Fatalf("request body = %#v", requestBody)
	}
}

// 验证 Anthropic Stream 会读取文本增量。
func TestAnthropicClientStreamReadsTextDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"he\"}}\n\n"))
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"llo\"}}\n\n"))
		_, _ = w.Write([]byte("event: message_stop\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer server.Close()

	client := infra.NewAnthropicLLMClient("sk-ant", "claude-sonnet-4-5", infra.WithAnthropicBaseURL(server.URL))
	stream, err := client.Stream(context.Background(), []*corellm.Message{corellm.UserMessage("say")})
	if err != nil {
		t.Fatalf("Stream error = %v", err)
	}
	var parts []string
	for token := range stream {
		parts = append(parts, token)
	}
	if got := strings.Join(parts, ""); got != "hello" {
		t.Fatalf("stream = %q, want hello", got)
	}
}

// 验证 Anthropic 原生工具流会保留文本增量并从 input_json_delta 聚合工具参数。
func TestAnthropicClientStreamWithToolsAggregatesToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"查询中\"}}\n\n"))
		_, _ = w.Write([]byte("event: content_block_start\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_1\",\"name\":\"search\",\"input\":{}}}\n\n"))
		_, _ = w.Write([]byte("event: content_block_delta\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"query\\\":\\\"golang\\\"}\"}}\n\n"))
		_, _ = w.Write([]byte("event: content_block_stop\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"content_block_stop\",\"index\":1}\n\n"))
		_, _ = w.Write([]byte("event: message_stop\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer server.Close()

	client := infra.NewAnthropicLLMClient("sk-ant", "claude-sonnet-4-5", infra.WithAnthropicBaseURL(server.URL))
	stream, err := client.(corellm.ToolStreamEventClient).StreamWithTools(context.Background(), []*corellm.Message{corellm.UserMessage("search")})
	if err != nil {
		t.Fatalf("StreamWithTools error = %v, want nil", err)
	}
	events := collectStreamEvents(stream)
	if len(events) != 3 || events[0].Kind != corellm.StreamEventTextDelta || events[0].Text != "查询中" {
		t.Fatalf("StreamWithTools events = %#v, want text/tool/done", events)
	}
	call := events[1].ToolCall
	if events[1].Kind != corellm.StreamEventToolCall || call == nil || call.ID != "toolu_1" || call.Name != "search" || call.Input["query"] != "golang" {
		t.Fatalf("StreamWithTools tool call = %#v, want aggregated search call", events[1])
	}
	if events[2].Kind != corellm.StreamEventDone {
		t.Fatalf("StreamWithTools final event = %#v, want done", events[2])
	}
}

// 验证 Anthropic Stream 会透传 TopP 参数。
func TestAnthropicClientStreamSendsTopP(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message_stop\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
	}))
	defer server.Close()

	client := infra.NewAnthropicLLMClient("sk-ant", "claude-sonnet-4-5", infra.WithAnthropicBaseURL(server.URL))
	stream, err := client.Stream(context.Background(), []*corellm.Message{corellm.UserMessage("say")}, corellm.WithTopP(0.6))
	if err != nil {
		t.Fatalf("Stream error = %v, want nil", err)
	}
	for range stream {
	}
	if requestBody["top_p"] != float64(0.6) {
		t.Fatalf("request body = %#v, want top_p 0.6", requestBody)
	}
}

// 验证 Anthropic embedding 当前仍返回不支持错误。
func TestAnthropicClientEmbeddingUnsupported(t *testing.T) {
	client := infra.NewAnthropicLLMClient("sk-ant", "claude-sonnet-4-5")
	if _, err := client.Embed(context.Background(), []string{"a"}, 0); err == nil || !strings.Contains(err.Error(), "不支持向量") {
		t.Fatalf("Embed error = %v", err)
	}
	if _, err := client.EmbedOne(context.Background(), "a", 0); err == nil || !strings.Contains(err.Error(), "不支持向量") {
		t.Fatalf("EmbedOne error = %v", err)
	}
}
