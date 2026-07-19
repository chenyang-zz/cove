package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultAnswer       = "Local chat reply persisted."
	defaultInitialDelay = 750 * time.Millisecond
	defaultChunkDelay   = 350 * time.Millisecond
)

type chatCompletionRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

type embeddingRequest struct {
	Model string          `json:"model"`
	Input json.RawMessage `json:"input"`
}

type server struct {
	answer       string
	initialDelay time.Duration
	chunkDelay   time.Duration
}

func main() {
	address := flag.String("address", envOrDefault("COVE_E2E_LLM_ADDRESS", "127.0.0.1:58001"), "listen address")
	answer := flag.String("answer", envOrDefault("COVE_E2E_LLM_ANSWER", defaultAnswer), "deterministic assistant answer")
	flag.Parse()

	initialDelay, err := envDuration("COVE_E2E_LLM_INITIAL_DELAY", defaultInitialDelay)
	if err != nil {
		log.Fatal(err)
	}
	chunkDelay, err := envDuration("COVE_E2E_LLM_CHUNK_DELAY", defaultChunkDelay)
	if err != nil {
		log.Fatal(err)
	}
	if strings.TrimSpace(*answer) == "" {
		log.Fatal("answer must not be empty")
	}

	handler := newHandler(server{
		answer:       strings.TrimSpace(*answer),
		initialDelay: initialDelay,
		chunkDelay:   chunkDelay,
	})
	log.Printf("deterministic OpenAI-compatible provider listening on %s", *address)
	if err := http.ListenAndServe(*address, handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func newHandler(provider server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /v1/chat/completions", provider.chatCompletions)
	mux.HandleFunc("POST /v1/embeddings", provider.embeddings)
	return mux
}

func (s server) embeddings(w http.ResponseWriter, request *http.Request) {
	var input embeddingRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, request.Body, 1<<20))
	if err := decoder.Decode(&input); err != nil {
		writeProviderError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if strings.TrimSpace(input.Model) == "" {
		writeProviderError(w, http.StatusBadRequest, "model is required")
		return
	}
	texts, err := embeddingInputs(input.Input)
	if err != nil {
		writeProviderError(w, http.StatusBadRequest, "input must be a non-empty string or string array")
		return
	}

	data := make([]map[string]any, 0, len(texts))
	for index, text := range texts {
		data = append(data, map[string]any{
			"object":    "embedding",
			"embedding": deterministicEmbedding(text),
			"index":     index,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object": "list",
		"data":   data,
		"model":  input.Model,
		"usage": map[string]int{
			"prompt_tokens": len(texts),
			"total_tokens":  len(texts),
		},
	})
}

func embeddingInputs(raw json.RawMessage) ([]string, error) {
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		if strings.TrimSpace(single) == "" {
			return nil, errors.New("input is empty")
		}
		return []string{single}, nil
	}

	var batch []string
	if err := json.Unmarshal(raw, &batch); err != nil || len(batch) == 0 {
		return nil, errors.New("input has unsupported shape")
	}
	for _, text := range batch {
		if strings.TrimSpace(text) == "" {
			return nil, errors.New("input contains an empty string")
		}
	}
	return batch, nil
}

func deterministicEmbedding(text string) []float64 {
	digest := sha256.Sum256([]byte(text))
	return []float64{
		float64(digest[0])/255*2 - 1,
		float64(digest[1])/255*2 - 1,
		float64(digest[2])/255*2 - 1,
	}
}

func writeProviderError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": message},
	})
}

func (s server) chatCompletions(w http.ResponseWriter, request *http.Request) {
	var input chatCompletionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, request.Body, 1<<20))
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, `{"error":{"message":"invalid JSON request"}}`, http.StatusBadRequest)
		return
	}
	if !input.Stream {
		http.Error(w, `{"error":{"message":"stream=true is required"}}`, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Model) == "" {
		http.Error(w, `{"error":{"message":"model is required"}}`, http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	if !wait(request, s.initialDelay) {
		return
	}
	for _, chunk := range splitAnswer(s.answer) {
		payload := map[string]any{
			"id":      "chatcmpl-cove-e2e",
			"object":  "chat.completion.chunk",
			"created": 1,
			"model":   input.Model,
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]string{"content": chunk},
			}},
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", encoded); err != nil {
			return
		}
		flusher.Flush()
		if !wait(request, s.chunkDelay) {
			return
		}
	}
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func splitAnswer(answer string) []string {
	runes := []rune(answer)
	if len(runes) < 3 {
		return []string{answer}
	}
	first := len(runes) / 3
	second := first * 2
	return []string{string(runes[:first]), string(runes[first:second]), string(runes[second:])}
}

func wait(request *http.Request, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-request.Context().Done():
		return false
	case <-timer.C:
		return true
	}
}

func envDuration(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative duration: %q", name, raw)
	}
	return value, nil
}

func envOrDefault(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
