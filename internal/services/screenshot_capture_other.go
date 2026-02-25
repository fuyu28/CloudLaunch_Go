//go:build !windows

package services

import "errors"

func captureWindowByPID(pid int, outputPath string) (CaptureMeta, error) {
	return CaptureMeta{}, errors.New("dxgi capture is not supported")
}
