//go:build linux

package app

import (
	"errors"
	"testing"
)

func TestOpenPathUsesXdgOpenAndPropagatesStartError(t *testing.T) {
	var gotName, gotPath string
	restore := setOpenPathStarterForTest(func(name, path string) error {
		gotName = name
		gotPath = path
		return errors.New("start failed")
	})
	t.Cleanup(restore)

	path := `/home/user/My Games/save data`
	err := openPath(path)
	if err == nil || err.Error() != "start failed" {
		t.Fatalf("openPath() error = %v, want start failed", err)
	}
	if gotName != "xdg-open" {
		t.Fatalf("command name = %q, want xdg-open", gotName)
	}
	if gotPath != path {
		t.Fatalf("command path = %q, want %q", gotPath, path)
	}
}
