//go:build !windows

// 非Windows向けのプロセス起動ヘルパ（隠しウィンドウ無し）。
package services

import (
	"context"
	"os/exec"
)

func execCommandHidden(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
