package documentparse

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fakeExtractor struct {
	text string
	err  error
}

func (f fakeExtractor) Extract(ctx context.Context, input Input) (string, error) {
	return f.text, f.err
}

type fakeDecoder struct {
	text string
	err  error
}

func (f fakeDecoder) Decode(data []byte) (string, error) {
	return f.text, f.err
}

func TestNewParserAppliesOptions(t *testing.T) {
	// 验证自定义文本解码器和扩展名 extractor 会覆盖默认行为。
	parser := NewParser(
		WithTextDecoder(fakeDecoder{text: "decoded text"}),
		WithExtractor(".custom", fakeExtractor{text: "custom text"}),
	)

	txt, err := parser.Parse(context.Background(), Input{Data: []byte("ignored"), FileExt: "txt"})
	if err != nil {
		t.Fatalf("Parse txt error = %v", err)
	}
	if txt.Text != "decoded text" || txt.FileExt != ".txt" {
		t.Fatalf("txt output = %+v, want decoded text with .txt", txt)
	}

	custom, err := parser.Parse(context.Background(), Input{Data: []byte("ignored"), FileExt: "custom"})
	if err != nil {
		t.Fatalf("Parse custom error = %v", err)
	}
	if custom.Text != "custom text" || custom.FileExt != ".custom" {
		t.Fatalf("custom output = %+v, want custom text with .custom", custom)
	}
}

func TestParseRejectsBlankAndUnsupportedInput(t *testing.T) {
	// 验证空白内容和未知扩展名会返回明确错误，避免上层把空文本继续入库。
	parser := NewParser()

	if _, err := parser.Parse(context.Background(), Input{Data: []byte("   \n\t"), FileExt: ".txt"}); err == nil {
		t.Fatal("Parse blank txt error = nil, want error")
	}
	if _, err := parser.Parse(context.Background(), Input{Data: []byte("data"), FileExt: ".exe"}); err == nil {
		t.Fatal("Parse unsupported ext error = nil, want error")
	}
}

func TestParseTextMarkdownHTMLAndDocx(t *testing.T) {
	// 验证默认解析器覆盖文本、Markdown、HTML 和 DOCX，并会规整多余空白。
	parser := NewParser()
	cases := []struct {
		name       string
		ext        string
		data       []byte
		wantHas    []string
		wantAbsent []string
	}{
		{name: "text", ext: "txt", data: []byte(" hello\nworld "), wantHas: []string{"hello", "world"}},
		{name: "markdown", ext: ".md", data: []byte("# Title\n\n**bold** text\n\n- [site](https://example.com)"), wantHas: []string{"Title", "bold", "text", "site"}, wantAbsent: []string{"<h1>", "https://example.com"}},
		{name: "html", ext: ".html", data: []byte(`<html><head><title>T</title><style>.x{}</style><script>x()</script></head><body><main>Hello <b>world</b></main><noscript>hidden</noscript><template>tpl</template><svg><text>vector</text></svg></body></html>`), wantHas: []string{"Hello", "world"}, wantAbsent: []string{"x()", ".x{}", "hidden", "tpl", "vector", "T"}},
		{name: "docx", ext: ".docx", data: minimalDocx(t, []string{"first paragraph", "second paragraph"}), wantHas: []string{"first paragraph", "second paragraph"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := parser.Parse(context.Background(), Input{Data: tc.data, FileExt: tc.ext})
			if err != nil {
				t.Fatalf("Parse error = %v", err)
			}
			for _, part := range tc.wantHas {
				if !strings.Contains(out.Text, part) {
					t.Fatalf("Parse text = %q, want contains %q", out.Text, part)
				}
			}
			for _, part := range tc.wantAbsent {
				if strings.Contains(out.Text, part) {
					t.Fatalf("Parse text = %q, want not contains %q", out.Text, part)
				}
			}
			if strings.Contains(out.Text, "  ") {
				t.Fatalf("Parse text = %q, want normalized spaces", out.Text)
			}
		})
	}
}

func TestParsePDFCanBeOverriddenAndDefaultInvalidPDFErrors(t *testing.T) {
	// 验证 PDF 默认解析失败可被调用方识别，同时允许通过 extractor 注入替换实现。
	parser := NewParser()
	out, err := parser.Parse(context.Background(), Input{Data: minimalPDF(), FileExt: ".pdf"})
	if err != nil {
		t.Fatalf("Parse valid pdf error = %v", err)
	}
	if !strings.Contains(out.Text, "Hello PDF") {
		t.Fatalf("Parse valid pdf text = %q, want contains Hello PDF", out.Text)
	}

	if _, err := parser.Parse(context.Background(), Input{Data: []byte("not a pdf"), FileExt: ".pdf"}); err == nil {
		t.Fatal("Parse invalid pdf error = nil, want error")
	}
	if _, err := parser.Parse(context.Background(), Input{Data: []byte("%PDF-1.4\n(fake text)"), FileExt: ".pdf"}); err == nil {
		t.Fatal("Parse fake pdf header error = nil, want third-party parser error")
	}

	overrideErr := errors.New("pdf extractor failed")
	parser = NewParser(WithExtractor(".pdf", fakeExtractor{err: overrideErr}))
	if _, err := parser.Parse(context.Background(), Input{Data: []byte("%PDF"), FileExt: ".pdf"}); !errors.Is(err, overrideErr) {
		t.Fatalf("Parse override pdf error = %v, want %v", err, overrideErr)
	}

	parser = NewParser(WithExtractor(".pdf", fakeExtractor{text: "pdf text"}))
	out, err = parser.Parse(context.Background(), Input{Data: []byte("%PDF"), FileExt: ".pdf"})
	if err != nil {
		t.Fatalf("Parse override pdf error = %v", err)
	}
	if out.Text != "pdf text" {
		t.Fatalf("Parse override pdf text = %q, want pdf text", out.Text)
	}
}

func minimalDocx(t *testing.T, paragraphs []string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create document.xml: %v", err)
	}
	var xml bytes.Buffer
	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, p := range paragraphs {
		xml.WriteString(`<w:p><w:r><w:t>`)
		xml.WriteString(p)
		xml.WriteString(`</w:t></w:r></w:p>`)
	}
	xml.WriteString(`</w:body></w:document>`)
	if _, err := w.Write(xml.Bytes()); err != nil {
		t.Fatalf("write document.xml: %v", err)
	}
	rels, err := zw.Create("word/_rels/document.xml.rels")
	if err != nil {
		t.Fatalf("create document.xml.rels: %v", err)
	}
	if _, err := rels.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`)); err != nil {
		t.Fatalf("write document.xml.rels: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func minimalPDF() []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	objects := []string{
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n",
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n",
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>\nendobj\n",
		"4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n",
		"5 0 obj\n<< /Length 44 >>\nstream\nBT /F1 24 Tf 100 700 Td (Hello PDF) Tj ET\nendstream\nendobj\n",
	}
	offsets := make([]int, 0, len(objects))
	for _, object := range objects {
		offsets = append(offsets, buf.Len())
		buf.WriteString(object)
	}
	startXref := buf.Len()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for _, offset := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}
	buf.WriteString("trailer\n<< /Root 1 0 R /Size 6 >>\n")
	buf.WriteString(fmt.Sprintf("startxref\n%d\n%%%%EOF", startXref))
	return buf.Bytes()
}
