package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScanLogicsDoesNotTreatServiceMethodsAsRouteLogic 验证集中式 Service 方法不会冒充路由专属 Logic。
func TestScanLogicsDoesNotTreatServiceMethodsAsRouteLogic(t *testing.T) {
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
	if _, ok := got[logicKey("gateway", "ListAccounts")]; ok {
		t.Fatalf("service method was indexed as route logic: %#v", got)
	}
}
