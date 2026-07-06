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
