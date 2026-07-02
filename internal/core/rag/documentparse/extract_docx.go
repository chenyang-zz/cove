package documentparse

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strings"

	docxlib "github.com/nguyenthenguyen/docx"
)

// extractDocx 使用 docx 库读取 OOXML 包，再把 document.xml 规整为纯文本。
func extractDocx(ctx context.Context, input Input) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	reader := bytes.NewReader(input.Data)
	doc, err := docxlib.ReadDocxFromMemory(reader, int64(len(input.Data)))
	if err != nil {
		return "", err
	}
	defer doc.Close()

	content := doc.Editable().GetContent()
	return extractDocxXML(strings.NewReader(content))
}

// extractDocxXML 只负责把 docx 库返回的 document.xml 文本节点转为纯文本。
func extractDocxXML(reader io.Reader) (string, error) {
	decoder := xml.NewDecoder(reader)
	var parts []string
	var current strings.Builder
	inText := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		switch tok := token.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "t":
				inText = true
			case "tab":
				current.WriteString(" ")
			case "br":
				current.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				current.WriteString(string(tok))
			}
		case xml.EndElement:
			switch tok.Name.Local {
			case "t":
				inText = false
			case "p":
				if text := strings.TrimSpace(current.String()); text != "" {
					parts = append(parts, text)
				}
				current.Reset()
			}
		}
	}
	if text := strings.TrimSpace(current.String()); text != "" {
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n"), nil
}
