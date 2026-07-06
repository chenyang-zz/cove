package llm

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	corellm "github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
	"github.com/boxify/api-go/internal/xerr"
)

const (
	defaultAnthropicBaseURL   = "https://api.anthropic.com"
	defaultAnthropicMaxTokens = int64(1024)
)

type anthropicLLMClient struct {
	client             anthropic.Client
	apiKey             string
	model              string
	defaultMaxTokens   int64
	defaultTemperature *float64
}

type anthropicConfig struct {
	httpClient         *http.Client
	apiKey             string
	baseURL            string
	defaultMaxTokens   int64
	defaultTemperature *float64
}

type AnthropicOption func(*anthropicConfig)

func WithAnthropicBaseURL(baseURL string) AnthropicOption {
	return func(c *anthropicConfig) {
		if strings.TrimSpace(baseURL) != "" {
			c.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		}
	}
}

func WithAnthropicMaxTokens(maxTokens int64) AnthropicOption {
	return func(c *anthropicConfig) {
		if maxTokens > 0 {
			c.defaultMaxTokens = maxTokens
		}
	}
}

func WithAnthropicTemperature(value float64) AnthropicOption {
	return func(c *anthropicConfig) {
		c.defaultTemperature = &value
	}
}

func NewAnthropicLLMClient(apiKey string, model string, opts ...AnthropicOption) corellm.Client {
	cfg := &anthropicConfig{
		httpClient:       &http.Client{Timeout: 60 * time.Second},
		apiKey:           strings.TrimSpace(apiKey),
		baseURL:          defaultAnthropicBaseURL,
		defaultMaxTokens: defaultAnthropicMaxTokens,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	clientOptions := []option.RequestOption{
		option.WithAPIKey(cfg.apiKey),
		option.WithBaseURL(cfg.baseURL),
		option.WithHTTPClient(cfg.httpClient),
	}
	return &anthropicLLMClient{
		client:             anthropic.NewClient(clientOptions...),
		apiKey:             cfg.apiKey,
		model:              strings.TrimSpace(model),
		defaultMaxTokens:   cfg.defaultMaxTokens,
		defaultTemperature: cfg.defaultTemperature,
	}
}

func (c *anthropicLLMClient) Invoke(ctx context.Context, messages []*corellm.Message, opts ...corellm.ModelCallOption) (string, error) {
	result, err := c.InvokeResult(ctx, messages, opts...)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

func (c *anthropicLLMClient) InvokeResult(ctx context.Context, messages []*corellm.Message, opts ...corellm.ModelCallOption) (*corellm.LLMResult, error) {
	if err := c.validateChatConfig(); err != nil {
		return nil, err
	}
	resp, err := c.client.Messages.New(ctx, c.messageParams(messages, opts...))
	if err != nil {
		return nil, xerr.Wrapf(err, "请求模型接口失败")
	}
	if resp == nil || len(resp.Content) == 0 {
		return nil, xerr.Internal("模型返回为空", nil)
	}
	var out strings.Builder
	toolCalls := make([]corellm.LLMToolCall, 0)
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				out.WriteString(block.Text)
			}
		case "tool_use":
			toolUse := block.AsToolUse()
			rawInput := string(toolUse.Input)
			toolCalls = append(toolCalls, corellm.LLMToolCall{
				ID:       toolUse.ID,
				Name:     toolUse.Name,
				Input:    parseToolInput(rawInput),
				RawInput: rawInput,
			})
		}
	}
	if out.Len() == 0 && len(toolCalls) == 0 {
		return nil, xerr.Internal("模型返回为空", nil)
	}
	usage := corellm.TokenUsage{
		InputTokens:              resp.Usage.InputTokens,
		OutputTokens:             resp.Usage.OutputTokens,
		CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
		CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
	}
	usage.TotalTokens = usage.InputTokens + usage.OutputTokens + usage.CacheCreationInputTokens + usage.CacheReadInputTokens
	return &corellm.LLMResult{
		Text:       out.String(),
		ToolCalls:  toolCalls,
		Model:      string(resp.Model),
		Provider:   "anthropic",
		ID:         resp.ID,
		StopReason: string(resp.StopReason),
		Usage:      usage,
		RawJSON:    resp.RawJSON(),
	}, nil
}

func (c *anthropicLLMClient) Stream(ctx context.Context, messages []*corellm.Message, opts ...corellm.ModelCallOption) (<-chan string, error) {
	if err := c.validateChatConfig(); err != nil {
		return nil, err
	}
	stream := c.client.Messages.NewStreaming(ctx, c.messageParams(messages, opts...))
	if err := stream.Err(); err != nil {
		return nil, xerr.Wrapf(err, "请求模型流式接口失败")
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		for stream.Next() {
			chunk := stream.Current()
			if chunk.Delta.Text == "" {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case ch <- chunk.Delta.Text:
			}
		}
	}()
	return ch, nil
}

func (c *anthropicLLMClient) Embed(context.Context, []string, int, ...corellm.EmbeddingOption) ([][]float64, error) {
	return nil, xerr.BadRequest("Anthropic 当前不支持向量模型调用")
}

func (c *anthropicLLMClient) EmbedOne(context.Context, string, int) ([]float64, error) {
	return nil, xerr.BadRequest("Anthropic 当前不支持向量模型调用")
}

func (c *anthropicLLMClient) messageParams(messages []*corellm.Message, opts ...corellm.ModelCallOption) anthropic.MessageNewParams {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: c.defaultMaxTokens,
	}
	if c.defaultTemperature != nil {
		opts = append([]corellm.ModelCallOption{corellm.WithTemperature(*c.defaultTemperature)}, opts...)
	}
	chatOpts := corellm.NewChatOptions(opts...)
	if chatOpts.Temperature != nil {
		params.Temperature = anthropic.Float(*chatOpts.Temperature)
	}
	if chatOpts.TopP != nil {
		params.TopP = anthropic.Float(*chatOpts.TopP)
	}
	if chatOpts.MaxTokens != nil && *chatOpts.MaxTokens > 0 {
		params.MaxTokens = *chatOpts.MaxTokens
	}
	params.Tools = toAnthropicTools(chatOpts.Tools)
	params.ToolChoice = toAnthropicToolChoice(chatOpts.ToolChoice)
	params.Messages, params.System = toAnthropicMessages(messages)
	return params
}

func toAnthropicTools(tools []coretool.Descriptor) []anthropic.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, item := range tools {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		toolParam := anthropic.ToolParam{
			Name:        strings.TrimSpace(item.Name),
			Description: anthropic.String(item.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties:  item.Schema.Parameters.Properties,
				Required:    item.Schema.Parameters.Required,
				ExtraFields: anthropicSchemaExtraFields(item.Schema.Parameters),
			},
		}
		if item.Schema.Strict != nil {
			toolParam.Strict = anthropic.Bool(*item.Schema.Strict)
		}
		out = append(out, anthropic.ToolUnionParam{OfTool: &toolParam})
	}
	return out
}

func toAnthropicToolChoice(choice *corellm.ToolChoice) anthropic.ToolChoiceUnionParam {
	if choice == nil {
		return anthropic.ToolChoiceUnionParam{}
	}
	switch choice.Mode {
	case corellm.ToolChoiceAuto:
		return anthropic.ToolChoiceUnionParam{OfAuto: &anthropic.ToolChoiceAutoParam{}}
	case corellm.ToolChoiceNone:
		none := anthropic.NewToolChoiceNoneParam()
		return anthropic.ToolChoiceUnionParam{OfNone: &none}
	case corellm.ToolChoiceRequired:
		return anthropic.ToolChoiceUnionParam{OfAny: &anthropic.ToolChoiceAnyParam{}}
	case corellm.ToolChoiceTool:
		if strings.TrimSpace(choice.Name) == "" {
			return anthropic.ToolChoiceUnionParam{}
		}
		return anthropic.ToolChoiceParamOfTool(strings.TrimSpace(choice.Name))
	default:
		return anthropic.ToolChoiceUnionParam{}
	}
}

func anthropicSchemaExtraFields(schema coretool.ParametersSchema) map[string]any {
	extra := map[string]any{}
	if schema.Type != "" && schema.Type != "object" {
		extra["type"] = schema.Type
	}
	if schema.AdditionalProperties != nil {
		extra["additionalProperties"] = schema.AdditionalProperties
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}

func (c *anthropicLLMClient) validateChatConfig() error {
	if c.apiKey == "" {
		return xerr.Internal("模型 API Key 未配置", nil)
	}
	if c.model == "" {
		return xerr.Internal("模型名称未配置", nil)
	}
	return nil
}

func toAnthropicMessages(messages []*corellm.Message) ([]anthropic.MessageParam, []anthropic.TextBlockParam) {
	out := make([]anthropic.MessageParam, 0, len(messages))
	system := make([]anthropic.TextBlockParam, 0)
	for _, m := range messages {
		if m == nil || strings.TrimSpace(m.Content) == "" {
			continue
		}
		switch m.Role {
		case corellm.SystemRole:
			system = append(system, anthropic.TextBlockParam{Text: m.Content})
		case corellm.AssistantRole:
			out = append(out, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		default:
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		}
	}
	return out, system
}
