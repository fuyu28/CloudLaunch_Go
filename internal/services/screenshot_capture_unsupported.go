//go:build !windows

// 非Windows向けスクリーンショット撮影のスタブ実装。
package services

import (
	"context"
	"errors"
)

// captureWithScreencap は非Windowsではサポート外。captureFunc がエラーを返すため、
// CaptureHotkey / CaptureGameScreenshot のオーケストレーション自体は共有ファイルで検証できる。
func (service *ScreenshotService) captureWithScreencap(ctx context.Context, pid int, outPath string) error {
	return errors.New("screenshot capture is only supported on Windows")
}
