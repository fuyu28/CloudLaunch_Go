//go:build windows

package services

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadPID  = user32.NewProc("GetWindowThreadProcessId")
)

func foregroundWindowAny() windows.Handle {
	fg, _, _ := procGetForegroundWindow.Call()
	if fg == 0 {
		return 0
	}
	return windows.Handle(fg)
}

func windowProcessID(hwnd windows.Handle) uint32 {
	if hwnd == 0 {
		return 0
	}
	var pid uint32
	procGetWindowThreadPID.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	return pid
}
