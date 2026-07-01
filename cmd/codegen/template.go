package main

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed templates/*.gotmpl
var templateFS embed.FS

func renderTemplate(name string, data any) (string, error) {
	tpl, err := template.ParseFS(templateFS, "templates/"+name)
	if err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	var out bytes.Buffer
	if err := tpl.ExecuteTemplate(&out, name, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return out.String(), nil
}
