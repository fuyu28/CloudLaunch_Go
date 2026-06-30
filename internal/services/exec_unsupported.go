//go:build !windows

package services

import (
	"context"
	"os/exec"
)

func execCommandHidden(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
