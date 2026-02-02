//go:build windows

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
