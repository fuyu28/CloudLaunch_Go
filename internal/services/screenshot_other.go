//go:build !windows

package services

import (
	"errors"
	"image"
)

func captureWindowImageByPID(pid int) (image.Image, error) {
	return nil, errors.New("スクリーンショットはWindowsのみ対応です")
}

func captureWindowWithWGC(pid int, outputPath string) (bool, error) {
	return false, nil
}
