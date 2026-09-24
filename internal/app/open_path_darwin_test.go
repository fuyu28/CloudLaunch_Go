//go:build darwin

package app

import (
	"errors"
	"testing"
)

func TestOpenPathUsesOpenAndPropagatesStartError(t *testing.T) {
	var gotName, gotPath string
	restore := setOpenPathStarterForTest(func(name, path string) error {
		gotName = name
		gotPath = path
		return errors.New("start failed")
	})
	t.Cleanup(restore)

	path := `/Users/テスト/Documents/セーブ`
	err := openPath(path)
	if err == nil || err.Error() != "start failed" {
		t.Fatalf("openPath() error = %v, want start failed", err)
	}
	if gotName != "open" {
		t.Fatalf("command name = %q, want open", gotName)
	}
	if gotPath != path {
		t.Fatalf("command path = %q, want %q", gotPath, path)
	}
}
