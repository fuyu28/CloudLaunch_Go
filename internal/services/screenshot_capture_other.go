//go:build !windows

package services

import (
	"context"
	"errors"
)

func (service *ScreenshotService) captureWithScreenClip(_ context.Context, _ string, _ string) error {
	return errors.New("screen clip is not supported")
}
