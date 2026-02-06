//go:build windows

package services

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func windowProcessID(hwnd windows.Handle) uint32 {
	if hwnd == 0 {
		return 0
	}
	var pid uint32
	procGetWindowThreadPID.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	return pid
}
