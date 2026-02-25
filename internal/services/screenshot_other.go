//go:build !windows

package services

func rankPidsForCapture(pids []int) []int {
	return pids
}
