package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScanLogicsRecognizesServiceMethods 验证集中式 Service 方法可满足路由 Logic 同步检查。
func TestScanLogicsRecognizesServiceMethods(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "logic", "gateway")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "service.go")
	if err := os.WriteFile(path, []byte("package gateway\ntype Service struct{}\nfunc (*Service) ListAccounts() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := scanLogics(root)
	if err != nil {
		t.Fatal(err)
	}
	if got[logicKey("gateway", "ListAccounts")] != path {
		t.Fatalf("service method was not indexed: %#v", got)
	}
}
