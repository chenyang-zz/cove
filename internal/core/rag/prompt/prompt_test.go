// Package prompt 验证 RAG prompt 的定义边界。
//
// 本文件确保 RAG prompt 只暴露模板文件、模板名和数据结构；模板读取和渲染必须通过
// core/prompt 完成。
package prompt

import (
	"strings"
	"testing"

	coreprompt "github.com/boxify/api-go/internal/core/prompt"
)

func TestContentClassifierTemplateCanBeRenderedByCorePrompt(t *testing.T) {
	// 验证 RAG prompt 只提供模板定义，模板渲染由 core/prompt 统一完成。
	out, err := coreprompt.Render(Templates, ContentClassifierTemplate, ContentClassifierData{
		Existing: "技术、学习",
		Content:  "这是一段内容",
	})
	if err != nil {
		t.Fatalf("core prompt Render error = %v", err)
	}
	if !strings.Contains(out, "技术、学习") || !strings.Contains(out, "这是一段内容") {
		t.Fatalf("rendered content classifier prompt = %q, want rendered data", out)
	}
	if strings.Contains(out, "{{") || strings.Contains(out, "}}") {
		t.Fatalf("rendered content classifier prompt = %q, want executed template", out)
	}
}

func TestImageDescriptionTemplateCanBeRenderedByCorePrompt(t *testing.T) {
	// 验证图片描述模板文件可由 core/prompt 读取渲染，rag/prompt 不承担解析职责。
	out, err := coreprompt.Render(Templates, ImageDescriptionTemplate, nil)
	if err != nil {
		t.Fatalf("core prompt Render error = %v", err)
	}
	if !strings.Contains(out, "description") || !strings.Contains(out, "ocr_text") {
		t.Fatalf("rendered image description prompt = %q, want structured image fields", out)
	}
}

func TestTemplateTextCanBeReadByCorePrompt(t *testing.T) {
	// 验证 RAG prompt 暴露模板文件系统和名称，原始模板文本读取由 core/prompt 提供。
	out, err := coreprompt.TemplateText(Templates, ContentClassifierTemplate)
	if err != nil {
		t.Fatalf("TemplateText error = %v", err)
	}
	if !strings.Contains(out, "{{ .Existing }}") || !strings.Contains(out, "{{ .Content }}") {
		t.Fatalf("TemplateText = %q, want raw classifier template placeholders", out)
	}
}
