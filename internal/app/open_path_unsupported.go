//go:build !windows && !darwin && !linux

// 未対応 OS 向けのパスオープンスタブ。
package app

import "errors"

var errOpenPathUnsupported = errors.New("opening paths is not supported on this platform")

func openPath(path string) error {
	return errOpenPathUnsupported
}
