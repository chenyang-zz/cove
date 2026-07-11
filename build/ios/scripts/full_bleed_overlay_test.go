package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPatchWebViewLayoutAddsKeyboardScrollLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webview_window_ios.m")
	upstream := bytes.Join([][]byte{
		viewControllerImplementation,
		viewDidLoadStart,
		scrollConfiguration,
		upstreamLayout,
	}, []byte("\n"))
	if err := os.WriteFile(path, upstream, 0o644); err != nil {
		t.Fatalf("write upstream fixture: %v", err)
	}

	patchWebViewLayout(path)

	patched, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read patched source: %v", err)
	}
	for _, expected := range [][]byte{
		[]byte("coveSetKeyboardScrollLocked"),
		[]byte("UIKeyboardWillChangeFrameNotification"),
		[]byte("CoveContentOffsetObservationContext"),
		fullBleedLayout,
	} {
		if !bytes.Contains(patched, expected) {
			t.Fatalf("patched source does not contain %q", expected)
		}
	}
}
