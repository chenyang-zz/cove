package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corellm "github.com/boxify/api-go/internal/core/llm"
	infra "github.com/boxify/api-go/internal/infrastructure/llm"
)

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

func TestAnthropicClientEmbeddingUnsupported(t *testing.T) {
	client := infra.NewAnthropicLLMClient("sk-ant", "claude-sonnet-4-5")
	if _, err := client.Embed(context.Background(), []string{"a"}, 0); err == nil || !strings.Contains(err.Error(), "不支持向量") {
		t.Fatalf("Embed error = %v", err)
	}
	if _, err := client.EmbedOne(context.Background(), "a", 0); err == nil || !strings.Contains(err.Error(), "不支持向量") {
		t.Fatalf("EmbedOne error = %v", err)
	}
}
