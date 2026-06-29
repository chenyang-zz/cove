package llm

import corellm "github.com/boxify/api-go/internal/core/llm"

type OpenAICompatibleFactory struct{}

func NewOpenAICompatibleFactory() OpenAICompatibleFactory {
	return OpenAICompatibleFactory{}
}

func (OpenAICompatibleFactory) NewClient(cfg corellm.ModelConfig) (corellm.Client, error) {
	opts := []OpenAIOption{WithBaseURL(cfg.BaseURL)}
	if cfg.EmbeddingModel != "" {
		opts = append(opts, WithEmbeddingModel(cfg.EmbeddingModel))
	}
	return NewOpenaiLLMClient(cfg.APIKey, cfg.Model, opts...), nil
}

type AnthropicFactory struct{}

func NewAnthropicFactory() AnthropicFactory {
	return AnthropicFactory{}
}

func (AnthropicFactory) NewClient(cfg corellm.ModelConfig) (corellm.Client, error) {
	opts := []AnthropicOption{}
	if cfg.BaseURL != "" {
		opts = append(opts, WithAnthropicBaseURL(cfg.BaseURL))
	}
	return NewAnthropicLLMClient(cfg.APIKey, cfg.Model, opts...), nil
}
