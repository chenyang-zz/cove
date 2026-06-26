/**
 * @Time   : 2026/6/23 01:26
 * @Author : chenyangzhao542@gmail.com
 * @File   : manager.go
 **/

package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/boxify/api-go/internal/xerr"
)

type Manager struct {
	fs            embed.FS
	MemoryPrompts *MemoryPrompts
}

func NewManager(fs embed.FS) *Manager {
	m := &Manager{
		fs: fs,
	}
	memoryPrompts := NewMemoryPrompts(m)

	m.MemoryPrompts = memoryPrompts
	return m
}

func (m *Manager) Render(name string, data any) (string, error) {
	path := fmt.Sprintf("prompts/%s.tmpl", name)

	content, err := m.fs.ReadFile(path)
	if err != nil {
		return "", xerr.Wrapf(err, "read prompt %s failed: %v", path, err)
	}

	tpl, err := template.New(name).Funcs(sprig.TxtFuncMap()).Parse(string(content))
	if err != nil {
		return "", xerr.Wrapf(err, "parse prompt %s failed: %v", name, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", xerr.Wrapf(err, "render prompt %s failed: %v", name, err)
	}

	return buf.String(), nil
}
