package llm

import (
	"context"
	"net/http"
	"strings"
	"time"

	corellm "github.com/boxify/api-go/internal/core/llm"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

type openaiLLMClient struct {
	client         openai.Client
	apiKey         string
	model          string
	embeddingModel string
}

type openAIConfig struct {
	httpClient     *http.Client
	apiKey         string
	baseURL        string
	embeddingModel string
}

type OpenAIOption func(*openAIConfig)

func WithBaseURL(baseURL string) OpenAIOption {
	return func(c *openAIConfig) {
		if strings.TrimSpace(baseURL) != "" {
			c.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		}
	}
}

func WithEmbeddingModel(model string) OpenAIOption {
	return func(c *openAIConfig) {
		if strings.TrimSpace(model) != "" {
			c.embeddingModel = strings.TrimSpace(model)
		}
	}
}

func NewOpenaiLLMClient(apiKey string, model string, opts ...OpenAIOption) corellm.Client {
	cfg := &openAIConfig{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		apiKey:     strings.TrimSpace(apiKey),
		baseURL:    defaultOpenAIBaseURL,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	model = strings.TrimSpace(model)
	if cfg.embeddingModel == "" {
		cfg.embeddingModel = model
	}

	clientOptions := []option.RequestOption{
		option.WithAPIKey(cfg.apiKey),
		option.WithBaseURL(cfg.baseURL),
		option.WithHTTPClient(cfg.httpClient),
	}
	return &openaiLLMClient{
		client:         openai.NewClient(clientOptions...),
		apiKey:         cfg.apiKey,
		model:          model,
		embeddingModel: cfg.embeddingModel,
	}
}

func (c *openaiLLMClient) Invoke(ctx context.Context, messages []*corellm.Message, opts ...corellm.ModelCallOption) (string, error) {
	if err := c.validateChatConfig(); err != nil {
		return "", err
	}
	resp, err := c.client.Chat.Completions.New(ctx, c.chatParams(messages, opts...))
	if err != nil {
		return "", xerr.Wrapf(err, "请求模型接口失败")
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", xerr.Internal("模型返回为空", nil)
	}
	return resp.Choices[0].Message.Content, nil
}

func (c *openaiLLMClient) Stream(ctx context.Context, messages []*corellm.Message, opts ...corellm.ModelCallOption) (<-chan string, error) {
	if err := c.validateChatConfig(); err != nil {
		return nil, err
	}
	stream := c.client.Chat.Completions.NewStreaming(ctx, c.chatParams(messages, opts...))
	if err := stream.Err(); err != nil {
		return nil, xerr.Wrapf(err, "请求模型流式接口失败")
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		for stream.Next() {
			chunk := stream.Current()
			for _, choice := range chunk.Choices {
				if choice.Delta.Content == "" {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case ch <- choice.Delta.Content:
				}
			}
		}
	}()
	return ch, nil
}

func (c *openaiLLMClient) chatParams(messages []*corellm.Message, opts ...corellm.ModelCallOption) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(c.model),
		Messages: toOpenAIMessages(messages),
	}
	chatOpts := corellm.NewChatOptions(opts...)
	if chatOpts.Temperature != nil {
		params.Temperature = openai.Float(*chatOpts.Temperature)
	}
	if chatOpts.MaxTokens != nil {
		params.MaxTokens = openai.Int(*chatOpts.MaxTokens)
	}
	return params
}

func (c *openaiLLMClient) Embed(ctx context.Context, texts []string, dimensions int) ([][]float64, error) {
	if err := c.validateEmbeddingConfig(); err != nil {
		return nil, err
	}
	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(c.embeddingModel),
		Input: openai.EmbeddingNewParamsInputUnion{OfArrayOfStrings: texts},
	}
	if dimensions > 0 {
		params.Dimensions = openai.Int(int64(dimensions))
	}

	resp, err := c.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, xerr.Wrapf(err, "请求模型向量接口失败")
	}
	if resp == nil {
		return nil, xerr.Internal("模型返回的向量为空", nil)
	}

	vecs := make([][]float64, 0, len(resp.Data))
	for _, item := range resp.Data {
		vec := make([]float64, 0, len(item.Embedding))
		for _, v := range item.Embedding {
			vec = append(vec, float64(v))
		}
		vecs = append(vecs, vec)
	}
	return vecs, nil
}

func (c *openaiLLMClient) EmbedOne(ctx context.Context, text string, dimensions int) ([]float64, error) {
	vecs, err := c.Embed(ctx, []string{text}, dimensions)
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, xerr.Internal("模型返回的向量为空", nil)
	}
	return vecs[0], nil
}

func (c *openaiLLMClient) validateChatConfig() error {
	if c.apiKey == "" {
		return xerr.Internal("模型 API Key 未配置", nil)
	}
	if c.model == "" {
		return xerr.Internal("模型名称未配置", nil)
	}
	return nil
}

func (c *openaiLLMClient) validateEmbeddingConfig() error {
	if c.apiKey == "" {
		return xerr.Internal("模型 API Key 未配置", nil)
	}
	if c.embeddingModel == "" {
		return xerr.Internal("向量模型名称未配置", nil)
	}
	return nil
}

func toOpenAIMessages(messages []*corellm.Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, m := range messages {
		if m == nil {
			continue
		}
		switch m.Role {
		case corellm.SystemRole:
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(m.Content),
					},
				},
			})
		case corellm.AssistantRole:
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(m.Content),
					},
				},
			})
		default:
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(m.Content),
					},
				},
			})
		}
	}
	return out
}
