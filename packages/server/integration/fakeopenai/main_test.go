package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestChatCompletionsStreamsDeterministicAnswer 验证兼容接口会以 SSE 分块返回确定性答案。
func TestChatCompletionsStreamsDeterministicAnswer(t *testing.T) {
	provider := httptest.NewServer(newHandler(server{answer: "deterministic answer"}))
	t.Cleanup(provider.Close)

	response, err := http.Post(
		provider.URL+"/v1/chat/completions",
		"application/json",
		bytes.NewBufferString(`{"model":"cove-e2e","stream":true}`),
	)
	if err != nil {
		t.Fatalf("POST chat completions error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 200; body=%q", response.StatusCode, body)
	}

	var streamed strings.Builder
	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") || line == "data: [DONE]" {
			continue
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk); err != nil {
			t.Fatalf("decode stream chunk error = %v", err)
		}
		if len(chunk.Choices) != 1 {
			t.Fatalf("choices = %d, want 1", len(chunk.Choices))
		}
		streamed.WriteString(chunk.Choices[0].Delta.Content)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan stream error = %v", err)
	}
	if streamed.String() != "deterministic answer" {
		t.Fatalf("streamed answer = %q, want deterministic answer", streamed.String())
	}
}

// TestEmbeddingsAcceptsStringAndArray 验证向量接口接受单字符串与字符串数组，并按输入顺序返回稳定的三维向量。
func TestEmbeddingsAcceptsStringAndArray(t *testing.T) {
	provider := httptest.NewServer(newHandler(server{answer: "unused"}))
	t.Cleanup(provider.Close)

	request := func(body string) []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} {
		t.Helper()
		response, err := http.Post(provider.URL+"/v1/embeddings", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("POST embeddings error = %v", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			responseBody, _ := io.ReadAll(response.Body)
			t.Fatalf("status = %d, want 200; body=%q", response.StatusCode, responseBody)
		}
		var decoded struct {
			Data []struct {
				Embedding []float64 `json:"embedding"`
				Index     int       `json:"index"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
			t.Fatalf("decode embeddings response error = %v", err)
		}
		return decoded.Data
	}

	single := request(`{"model":"cove-e2e-embedding","input":"alpha"}`)
	batch := request(`{"model":"cove-e2e-embedding","input":["alpha","beta"]}`)
	secondBatch := request(`{"model":"cove-e2e-embedding","input":["alpha","beta"]}`)
	if len(single) != 1 || len(batch) != 2 || len(secondBatch) != 2 {
		t.Fatalf("embedding counts single=%d batch=%d second_batch=%d, want 1, 2, 2", len(single), len(batch), len(secondBatch))
	}
	if len(single[0].Embedding) != 3 || len(batch[0].Embedding) != 3 || len(batch[1].Embedding) != 3 {
		t.Fatalf("embedding dimensions single=%d batch=[%d,%d], want all 3", len(single[0].Embedding), len(batch[0].Embedding), len(batch[1].Embedding))
	}
	if single[0].Index != 0 || batch[0].Index != 0 || batch[1].Index != 1 {
		t.Fatalf("embedding indices single=%d batch=[%d,%d], want 0 and [0,1]", single[0].Index, batch[0].Index, batch[1].Index)
	}
	if fmt.Sprint(single[0].Embedding) != fmt.Sprint(batch[0].Embedding) || fmt.Sprint(batch) != fmt.Sprint(secondBatch) {
		t.Fatalf("embeddings are not stable: single=%v batch=%v second_batch=%v", single, batch, secondBatch)
	}
	if fmt.Sprint(batch[0].Embedding) == fmt.Sprint(batch[1].Embedding) {
		t.Fatalf("distinct inputs returned identical embeddings: %v", batch)
	}
}

// TestEmbeddingsRejectsUnsafeInputShapes 验证向量接口对畸形、缺字段或非字符串输入返回不泄露请求内容的安全错误。
func TestEmbeddingsRejectsUnsafeInputShapes(t *testing.T) {
	provider := httptest.NewServer(newHandler(server{answer: "unused"}))
	t.Cleanup(provider.Close)

	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"model":`},
		{name: "missing model", body: `{"input":"private payload"}`},
		{name: "number input", body: `{"model":"cove-e2e-embedding","input":42}`},
		{name: "mixed array", body: `{"model":"cove-e2e-embedding","input":["ok",42]}`},
		{name: "empty array", body: `{"model":"cove-e2e-embedding","input":[]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := http.Post(provider.URL+"/v1/embeddings", "application/json", bytes.NewBufferString(test.body))
			if err != nil {
				t.Fatalf("POST embeddings error = %v", err)
			}
			defer response.Body.Close()
			responseBody, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("ReadAll embeddings error response = %v", err)
			}
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%q", response.StatusCode, responseBody)
			}
			if strings.Contains(string(responseBody), "private payload") || strings.Contains(string(responseBody), "42") {
				t.Fatalf("error response leaked request content: %q", responseBody)
			}
		})
	}
}
