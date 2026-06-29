package llm

import (
	"strings"

	"github.com/boxify/api-go/internal/xerr"
)

type ModelConfig struct {
	Provider       string
	Model          string
	APIKey         string
	BaseURL        string
	EmbeddingModel string
}

type Factory interface {
	NewClient(cfg ModelConfig) (Client, error)
}

type Manager struct {
	factories map[string]Factory
}

func NewManager() *Manager {
	return &Manager{factories: map[string]Factory{}}
}

func (m *Manager) Register(provider string, factory Factory) {
	if m == nil || factory == nil {
		return
	}
	provider = normalizeProvider(provider)
	if provider == "" {
		return
	}
	m.factories[provider] = factory
}

func (m *Manager) ClientFor(cfg ModelConfig) (Client, error) {
	return m.NewClient(cfg)
}

func (m *Manager) NewClient(cfg ModelConfig) (Client, error) {
	cfg.Provider = normalizeProvider(cfg.Provider)
	cfg.Model = strings.TrimSpace(cfg.Model)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.EmbeddingModel = strings.TrimSpace(cfg.EmbeddingModel)

	if cfg.Provider == "" {
		return nil, xerr.BadRequest("模型 provider 未配置")
	}
	if cfg.Model == "" {
		return nil, xerr.BadRequest("模型名称未配置")
	}
	if cfg.APIKey == "" {
		return nil, xerr.BadRequest("模型 API Key 未配置")
	}
	if m == nil {
		return nil, xerr.BadRequestf("不支持的模型 provider: %s", cfg.Provider)
	}
	factory, ok := m.factories[cfg.Provider]
	if !ok {
		return nil, xerr.BadRequestf("不支持的模型 provider: %s", cfg.Provider)
	}
	return factory.NewClient(cfg)
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
