package app

import (
	"testing"
)

func TestOpenPathCommandPreservesPathArgument(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{name: "windows whitespace", path: `C:\Games\My Game\save`},
		{name: "japanese path", path: `/Users/山田/Documents/セーブデータ`},
		{name: "unix whitespace", path: `/tmp/path with spaces/dir`},
		{name: "mixed", path: `D:\ゲーム\Save Data\スロット 1`},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			command := openPathCommand("explorer", test.path)
			if len(command.Args) != 2 {
				t.Fatalf("Args = %#v, want exactly 2 elements", command.Args)
			}
			if command.Args[0] != "explorer" {
				t.Fatalf("Args[0] = %q, want explorer", command.Args[0])
			}
			if command.Args[1] != test.path {
				t.Fatalf("Args[1] = %q, want %q", command.Args[1], test.path)
			}
		})
	}
}

func TestStartOpenPathCommandStartupError(t *testing.T) {
	t.Parallel()

	err := startOpenPathCommand("cloudlaunch-open-path-missing-binary", "/tmp/does-not-matter")
	if err == nil {
		t.Fatal("expected startup error for missing binary")
	}
}
