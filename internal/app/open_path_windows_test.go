//go:build windows

package app

import (
	"errors"
	"testing"
)

func TestOpenPathUsesExplorerAndPropagatesStartError(t *testing.T) {
	var gotName, gotPath string
	restore := setOpenPathStarterForTest(func(name, path string) error {
		gotName = name
		gotPath = path
		return errors.New("start failed")
	})
	t.Cleanup(restore)

	path := `C:\Users\テスト\My Folder`
	err := openPath(path)
	if err == nil || err.Error() != "start failed" {
		t.Fatalf("openPath() error = %v, want start failed", err)
	}
	if gotName != "explorer" {
		t.Fatalf("command name = %q, want explorer", gotName)
	}
	if gotPath != path {
		t.Fatalf("command path = %q, want %q", gotPath, path)
	}
}
