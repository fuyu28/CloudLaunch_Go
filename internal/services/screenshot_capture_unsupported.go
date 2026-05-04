//go:build !windows

package services

import (
	"context"
	"errors"
)

func (service *ScreenshotService) CaptureHotkey(ctx context.Context, preferredGameID string) (string, string, error) {
	return "", "", errors.New("screenshot capture is only supported on Windows")
}

func (service *ScreenshotService) captureWithScreenClip(ctx context.Context, fullPath string, tmpPath string) error {
	return errors.New("screenshot capture is only supported on Windows")
}
