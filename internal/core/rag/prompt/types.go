// Package prompt 定义 RAG 默认提示词模板资源和模板参数结构。
//
// 本包只声明模板文件和模板变量，不读取、不解析、不渲染模板。调用方需要通过
// internal/core/prompt 包完成模板管理。
//
// 核心定义示例：
//
// Templates 配合 core/prompt 读取模板原文：
//
//	text := coreprompt.MustTemplateText(prompt.Templates, prompt.ContentClassifierTemplate)
//
// Templates 配合 core/prompt 渲染模板：
//
//	out, err := coreprompt.Render(prompt.Templates, prompt.ContentClassifierTemplate, data)
//
// ContentClassifierData 约束 content_classifier.tmpl 可使用的变量：
//
//	data := prompt.ContentClassifierData{Existing: "技术、生活", Content: "待分类文本"}
package prompt

import "embed"

// Templates 暴露 RAG 默认提示词模板文件，具体读取和渲染由 core/prompt 负责。
//
//go:embed *.tmpl
var Templates embed.FS

const (
	// ContentClassifierTemplate 是文本分类提示词模板文件名。
	ContentClassifierTemplate = "content_classifier.tmpl"
	// ImageDescriptionTemplate 是图片描述提示词模板文件名。
	ImageDescriptionTemplate = "image_description.tmpl"
)

// ContentClassifierData 约束文本分类默认模板可使用的变量。
type ContentClassifierData struct {
	// Existing 是调用方已有标签列表拼接后的文本。
	Existing string
	// Content 是裁剪后的待分类正文。
	Content string
}
