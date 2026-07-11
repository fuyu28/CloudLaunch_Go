//go:build windows

// Windows向けの隠しウィンドウ付きプロセス起動ヘルパ。
package services

import (
	"context"
	"os/exec"
	"syscall"
)

func execCommandHidden(ctx context.Context, name string, args ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, name, args...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return command
}
