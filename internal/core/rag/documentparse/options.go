package documentparse

import (
	"context"
	"strings"
)

// Options 表示 Parser 的长期配置。
type Options struct {
	Extractors  map[string]Extractor
	TextDecoder TextDecoder
}

// Option 用于调整 Parser 的长期配置。
type Option func(*Options)

// WithExtractor 为指定扩展名注册或覆盖文本提取器。
func WithExtractor(fileExt string, extractor Extractor) Option {
	return func(opts *Options) {
		ext := normalizeExt(fileExt)
		if ext != "" && extractor != nil {
			opts.Extractors[ext] = extractor
		}
	}
}

// WithTextDecoder 覆盖纯文本默认提取器使用的字节解码器。
func WithTextDecoder(decoder TextDecoder) Option {
	return func(opts *Options) {
		if decoder != nil {
			opts.TextDecoder = decoder
		}
	}
}

// defaultExtractors 构造内置文件格式提取器集合。
func defaultExtractors(decoder TextDecoder) map[string]Extractor {
	text := extractorFunc(func(ctx context.Context, input Input) (string, error) {
		return decoder.Decode(input.Data)
	})
	html := extractorFunc(extractHTML)
	markdown := extractorFunc(extractMarkdown)
	return map[string]Extractor{
		".txt":      text,
		".md":       markdown,
		".markdown": markdown,
		".html":     html,
		".htm":      html,
		".docx":     extractorFunc(extractDocx),
		".pdf":      extractorFunc(extractPDF),
	}
}

// normalizeExt 规整文件扩展名，确保以小写点号开头。
func normalizeExt(fileExt string) string {
	ext := strings.ToLower(strings.TrimSpace(fileExt))
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}
