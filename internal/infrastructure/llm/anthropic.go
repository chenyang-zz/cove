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

func (c *anthropicLLMClient) InvokeWithTools(ctx context.Context, messages []*corellm.Message, opts ...corellm.ModelCallOption) (*corellm.LLMResult, error) {
	return c.InvokeResult(ctx, messages, opts...)
}

func (c *anthropicLLMClient) Stream(ctx context.Context, messages []*corellm.Message, opts ...corellm.ModelCallOption) (<-chan string, error) {
	events, err := c.StreamEvents(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}
	ch := make(chan string)
	go func() {
		defer close(ch)
		for event := range events {
			if event.Kind != corellm.StreamEventTextDelta || event.Text == "" {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case ch <- event.Text:
			}
		}
	}()
	return ch, nil
}

// StreamEvents 执行结构化流式文本生成。
func (c *anthropicLLMClient) StreamEvents(ctx context.Context, messages []*corellm.Message, opts ...corellm.ModelCallOption) (<-chan corellm.StreamEvent, error) {
	return c.streamEvents(ctx, messages, opts...)
}

// StreamWithTools 执行支持原生工具调用的结构化流式生成。
func (c *anthropicLLMClient) StreamWithTools(ctx context.Context, messages []*corellm.Message, opts ...corellm.ModelCallOption) (<-chan corellm.StreamEvent, error) {
	return c.streamEvents(ctx, messages, opts...)
}

func (c *anthropicLLMClient) streamEvents(ctx context.Context, messages []*corellm.Message, opts ...corellm.ModelCallOption) (<-chan corellm.StreamEvent, error) {
	if err := c.validateChatConfig(); err != nil {
		return nil, err
	}
	stream := c.client.Messages.NewStreaming(ctx, c.messageParams(messages, opts...))
	if err := stream.Err(); err != nil {
		return nil, xerr.Wrapf(err, "请求模型流式接口失败")
	}

	ch := make(chan corellm.StreamEvent)
	go func() {
		defer close(ch)
		toolCalls := make(map[int64]*anthropicStreamToolCall)
		for stream.Next() {
			event := stream.Current()
			switch value := event.AsAny().(type) {
			case anthropic.ContentBlockStartEvent:
				if block, ok := value.ContentBlock.AsAny().(anthropic.ToolUseBlock); ok {
					rawInput := string(block.Input)
					// Claude 的细粒度工具流会先发送空对象，再通过 input_json_delta 补齐参数。
					if rawInput == "{}" {
						rawInput = ""
					}
					toolCalls[value.Index] = &anthropicStreamToolCall{ID: block.ID, Name: block.Name, Arguments: rawInput}
				}
			case anthropic.ContentBlockDeltaEvent:
				switch delta := value.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					if !sendStreamEvent(ctx, ch, corellm.StreamEvent{Kind: corellm.StreamEventTextDelta, Text: delta.Text}) {
						return
					}
				case anthropic.InputJSONDelta:
					if call := toolCalls[value.Index]; call != nil {
						call.Arguments += delta.PartialJSON
					}
				}
			case anthropic.ContentBlockStopEvent:
				call := toolCalls[value.Index]
				if call == nil {
					continue
				}
				delete(toolCalls, value.Index)
				if !sendStreamEvent(ctx, ch, corellm.StreamEvent{Kind: corellm.StreamEventToolCall, ToolCall: &corellm.LLMToolCall{
					ID:       call.ID,
					Name:     call.Name,
					Input:    parseToolInput(call.Arguments),
					RawInput: call.Arguments,
				}}) {
					return
				}
			case anthropic.MessageStopEvent:
				sendStreamEvent(ctx, ch, corellm.StreamEvent{Kind: corellm.StreamEventDone})
				return
			}
		}
		if err := stream.Err(); err != nil {
			sendStreamEvent(ctx, ch, corellm.StreamEvent{Kind: corellm.StreamEventError, Err: xerr.Wrapf(err, "请求模型流式接口失败")})
			return
		}
		sendStreamEvent(ctx, ch, corellm.StreamEvent{Kind: corellm.StreamEventDone})
	}()
	return ch, nil
}

type anthropicStreamToolCall struct {
	ID        string
	Name      string
	Arguments string
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
	extra := schema.Map()
	delete(extra, "properties")
	delete(extra, "required")
	if extra["type"] == "object" {
		delete(extra, "type")
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
		if m == nil {
			continue
		}
		if strings.TrimSpace(m.Content) == "" && len(m.ToolCalls) == 0 && m.Role != corellm.ToolRole {
			continue
		}
		switch m.Role {
		case corellm.SystemRole:
			system = append(system, anthropic.TextBlockParam{Text: m.Content})
		case corellm.AssistantRole:
			blocks := make([]anthropic.ContentBlockParamUnion, 0, len(m.ToolCalls)+1)
			if m.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(m.Content))
			}
			for _, call := range m.ToolCalls {
				name := strings.TrimSpace(call.Name)
				id := strings.TrimSpace(call.ID)
				if name == "" || id == "" {
					continue
				}
				blocks = append(blocks, anthropic.NewToolUseBlock(id, anthropicToolInput(call), name))
			}
			if len(blocks) > 0 {
				out = append(out, anthropic.NewAssistantMessage(blocks...))
			}
		case corellm.ToolRole:
			if strings.TrimSpace(m.ToolCallID) == "" {
				continue
			}
			out = append(out, anthropic.NewUserMessage(anthropic.NewToolResultBlock(strings.TrimSpace(m.ToolCallID), m.Content, false)))
		default:
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		}
	}
	return out, system
}

func anthropicToolInput(call corellm.LLMToolCall) any {
	if len(call.Input) > 0 {
		return call.Input
	}
	if input := parseToolInput(call.RawInput); len(input) > 0 {
		return input
	}
	return map[string]any{}
}

var _ corellm.ToolCallingClient = (*anthropicLLMClient)(nil)
