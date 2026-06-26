package llm

type ModelCallOptions struct {
	Temperature *float64
	MaxTokens   *int64
}

type ModelCallOption func(*ModelCallOptions)

func NewChatOptions(opts ...ModelCallOption) ModelCallOptions {
	var out ModelCallOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&out)
		}
	}
	return out
}

func WithTemperature(value float64) ModelCallOption {
	return func(opts *ModelCallOptions) {
		opts.Temperature = &value
	}
}

func WithMaxTokens(value int64) ModelCallOption {
	return func(opts *ModelCallOptions) {
		opts.MaxTokens = &value
	}
}
