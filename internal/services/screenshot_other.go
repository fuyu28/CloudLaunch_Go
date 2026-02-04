//go:build !windows

package services

import (
	"errors"
	"image"
)

func captureWindowImageByPID(pid int, clientOnly bool) (image.Image, error) {
	return nil, errors.New("スクリーンショットはWindowsのみ対応です")
}

func captureWindowWithWGC(pid int, outputPath string, clientOnly bool) (bool, error) {
	return false, nil
}
