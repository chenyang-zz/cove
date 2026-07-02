package documentparse

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

var (
	errEmptyContent    = errors.New("document content is empty")
	errUnsupportedType = errors.New("unsupported document file type")
)

// Parser 根据文件扩展名把文档二进制解析为纯文本。
type Parser struct {
	Options
}

type utf8Decoder struct{}

// NewParser 创建文档解析器，并初始化默认提取器。
//
// opts 会覆盖默认配置。自定义 extractor 会覆盖同扩展名的默认 extractor。
func NewParser(opts ...Option) *Parser {
	decoder := utf8Decoder{}
	parser := &Parser{
		Options: Options{
			TextDecoder: decoder,
			Extractors:  map[string]Extractor{},
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&parser.Options)
		}
	}
	customExtractors := parser.Extractors
	parser.Extractors = defaultExtractors(parser.TextDecoder)
	for ext, extractor := range customExtractors {
		parser.Extractors[ext] = extractor
	}
	return parser
}

// Parse 按文件扩展名解析文档文本。
//
// input.FileExt 为空或未注册时返回不支持类型错误。提取器返回空文本时返回
// errEmptyContent。ctx 会传递给具体 extractor，用于后续支持取消感知的实现。
func (p *Parser) Parse(ctx context.Context, input Input) (*Output, error) {
	if p == nil {
		return nil, errors.New("rag document parser is nil")
	}
	ext := normalizeExt(input.FileExt)
	extractor, ok := p.Extractors[ext]
	if ext == "" || !ok {
		return nil, fmt.Errorf("%w: %s", errUnsupportedType, ext)
	}

	// extractor 只负责格式解析，统一在出口规整空白和拦截空文本。
	text, err := extractor.Extract(ctx, Input{Data: input.Data, FileExt: ext})
	if err != nil {
		return nil, err
	}
	text = normalizeText(text)
	if text == "" {
		return nil, errEmptyContent
	}
	return &Output{Text: text, FileExt: ext}, nil
}

// Decode 把文本字节解码为 UTF-8 字符串，并替换非法 UTF-8 字节。
func (utf8Decoder) Decode(data []byte) (string, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return "", errEmptyContent
	}
	if utf8.Valid(data) {
		return string(data), nil
	}
	return string(bytes.ToValidUTF8(data, []byte(" "))), nil
}

// normalizeText 规整空白。
func normalizeText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
