//go:build !windows

// Windows 以外でプロセス起動ヘルパの互換実装を提供する。
package services

import (
	"context"
	"os/exec"
)

func execCommandHidden(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
