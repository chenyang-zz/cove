package config_test

import (
	"path/filepath"
	"testing"

	"github.com/boxify/api-go/internal/config"
)

// TestGatewayConfigDefaultsDisabled 验证网关默认关闭且安全边界有明确上限。
func TestGatewayConfigDefaultsDisabled(t *testing.T) {
	cfg, err := config.LoadFile(filepath.Join(t.TempDir(), "missing.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Gateway.Enabled || cfg.GatewayAddr() != "0.0.0.0:8010" {
		t.Fatalf("unexpected gateway defaults: %#v", cfg.Gateway)
	}
	if cfg.Gateway.MaxRequestBytes != 1<<20 || cfg.Gateway.MaxMediaBytes != 20<<20 {
		t.Fatalf("unexpected gateway limits: %#v", cfg.Gateway)
	}
}

// TestGatewayConfigEnvOverrides 验证独立网关可通过部署环境变量启用和调参。
func TestGatewayConfigEnvOverrides(t *testing.T) {
	t.Setenv("GATEWAY_ENABLED", "true")
	t.Setenv("GATEWAY_HOST", "127.0.0.1")
	t.Setenv("GATEWAY_PORT", "9010")
	t.Setenv("GATEWAY_MAX_MEDIA_BYTES", "4096")
	cfg, err := config.LoadFile(filepath.Join(t.TempDir(), "missing.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Gateway.Enabled || cfg.GatewayAddr() != "127.0.0.1:9010" || cfg.Gateway.MaxMediaBytes != 4096 {
		t.Fatalf("gateway env overrides failed: %#v", cfg.Gateway)
	}
}
