//go:build windows

package services

import (
	"errors"
	"image"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	biRGB                  = 0
	dibRGBColors           = 0
	srccopy                = 0x00CC0020
	pwClientOnly           = 0x00000001
	dwmExtendedFrameBounds = 9
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	gdi32                  = windows.NewLazySystemDLL("gdi32.dll")
	dwmapi                 = windows.NewLazySystemDLL("dwmapi.dll")
	procEnumWindows        = user32.NewProc("EnumWindows")
	procGetWindowThreadPID = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible    = user32.NewProc("IsWindowVisible")
	procIsIconic           = user32.NewProc("IsIconic")
	procGetWindowRect      = user32.NewProc("GetWindowRect")
	procGetClientRect      = user32.NewProc("GetClientRect")
	procClientToScreen     = user32.NewProc("ClientToScreen")
	procGetDC              = user32.NewProc("GetDC")
	procGetWindowDC        = user32.NewProc("GetWindowDC")
	procReleaseDC          = user32.NewProc("ReleaseDC")
	procPrintWindow        = user32.NewProc("PrintWindow")
	procDwmGetWindowAttr   = dwmapi.NewProc("DwmGetWindowAttribute")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateBitmap       = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procBitBlt             = gdi32.NewProc("BitBlt")
	procGetDIBits          = gdi32.NewProc("GetDIBits")
)

type windowRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

func captureWindowImageByPID(pid int) (image.Image, error) {
	hwnd, err := findBestWindowForPID(uint32(pid))
	if err != nil {
		return nil, err
	}

	if img, err := captureWindowWithPrintWindow(hwnd, true); err == nil && img != nil {
		return img, nil
	}

	if img, err := captureWindowWithPrintWindow(hwnd, false); err == nil && img != nil {
		if trimmed, trimErr := trimWithDwmBounds(hwnd, img); trimErr == nil && trimmed != nil {
			return trimmed, nil
		}
		return img, nil
	}

	return captureWindowClientWithBitBlt(hwnd)
}

func captureWindowWithPrintWindow(hwnd windows.Handle, clientOnly bool) (image.Image, error) {
	var rect windowRect
	if clientOnly {
		if ret, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect))); ret == 0 {
			return nil, errors.New("failed to get client rect")
		}
	} else {
		if ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect))); ret == 0 {
			return nil, errors.New("failed to get window rect")
		}
	}

	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	if width <= 0 || height <= 0 {
		return nil, errors.New("invalid window size")
	}

	hdcWindow, _, _ := procGetWindowDC.Call(uintptr(hwnd))
	if hdcWindow == 0 {
		return nil, errors.New("failed to get window DC")
	}
	defer procReleaseDC.Call(uintptr(hwnd), hdcWindow)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcWindow)
	if hdcMem == 0 {
		return nil, errors.New("failed to create compatible DC")
	}
	defer procDeleteDC.Call(hdcMem)

	hBitmap, _, _ := procCreateBitmap.Call(hdcWindow, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		return nil, errors.New("failed to create bitmap")
	}
	defer procDeleteObject.Call(hBitmap)

	oldObj, _, _ := procSelectObject.Call(hdcMem, hBitmap)
	defer procSelectObject.Call(hdcMem, oldObj)

	flags := uintptr(0)
	if clientOnly {
		flags = pwClientOnly
	}
	printResult, _, _ := procPrintWindow.Call(uintptr(hwnd), hdcMem, flags)
	if printResult == 0 {
		return nil, errors.New("PrintWindow failed")
	}

	return bitmapToImage(hdcMem, hBitmap, width, height)
}

func captureWindowClientWithBitBlt(hwnd windows.Handle) (image.Image, error) {
	var rect windowRect
	if ret, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect))); ret == 0 {
		return nil, errors.New("failed to get client rect")
	}
	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)
	if width <= 0 || height <= 0 {
		return nil, errors.New("invalid client size")
	}

	hdcWindow, _, _ := procGetDC.Call(uintptr(hwnd))
	if hdcWindow == 0 {
		return nil, errors.New("failed to get client DC")
	}
	defer procReleaseDC.Call(uintptr(hwnd), hdcWindow)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcWindow)
	if hdcMem == 0 {
		return nil, errors.New("failed to create compatible DC")
	}
	defer procDeleteDC.Call(hdcMem)

	hBitmap, _, _ := procCreateBitmap.Call(hdcWindow, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		return nil, errors.New("failed to create bitmap")
	}
	defer procDeleteObject.Call(hBitmap)

	oldObj, _, _ := procSelectObject.Call(hdcMem, hBitmap)
	defer procSelectObject.Call(hdcMem, oldObj)

	bitbltResult, _, _ := procBitBlt.Call(
		hdcMem,
		0,
		0,
		uintptr(width),
		uintptr(height),
		hdcWindow,
		0,
		0,
		srccopy,
	)
	if bitbltResult == 0 {
		return nil, errors.New("failed to capture client")
	}

	return bitmapToImage(hdcMem, hBitmap, width, height)
}

func bitmapToImage(hdcMem uintptr, hBitmap uintptr, width int, height int) (image.Image, error) {
	bmi := bitmapInfo{}
	bmi.Header.Size = uint32(unsafe.Sizeof(bmi.Header))
	bmi.Header.Width = int32(width)
	bmi.Header.Height = -int32(height)
	bmi.Header.Planes = 1
	bmi.Header.BitCount = 32
	bmi.Header.Compression = biRGB

	buf := make([]byte, width*height*4)
	ret, _, _ := procGetDIBits.Call(
		hdcMem,
		hBitmap,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bmi)),
		dibRGBColors,
	)
	if ret == 0 {
		return nil, errors.New("failed to read bitmap")
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < len(buf); i += 4 {
		b := buf[i]
		g := buf[i+1]
		r := buf[i+2]
		a := buf[i+3]
		img.Pix[i] = r
		img.Pix[i+1] = g
		img.Pix[i+2] = b
		img.Pix[i+3] = a
	}
	return img, nil
}

func trimWithDwmBounds(hwnd windows.Handle, img image.Image) (image.Image, error) {
	var frame windowRect
	ret, _, _ := procDwmGetWindowAttr.Call(
		uintptr(hwnd),
		uintptr(dwmExtendedFrameBounds),
		uintptr(unsafe.Pointer(&frame)),
		unsafe.Sizeof(frame),
	)
	if ret != 0 {
		return nil, errors.New("failed to get DWM bounds")
	}

	var window windowRect
	if ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&window))); ret == 0 {
		return nil, errors.New("failed to get window rect")
	}

	left := int(frame.Left - window.Left)
	top := int(frame.Top - window.Top)
	right := int(frame.Right - window.Left)
	bottom := int(frame.Bottom - window.Top)

	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	bounds := img.Bounds()
	if right > bounds.Dx() {
		right = bounds.Dx()
	}
	if bottom > bounds.Dy() {
		bottom = bounds.Dy()
	}
	if right <= left || bottom <= top {
		return nil, errors.New("invalid DWM bounds")
	}

	crop := image.Rect(left, top, right, bottom)
	out := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	for y := 0; y < crop.Dy(); y++ {
		for x := 0; x < crop.Dx(); x++ {
			out.Set(x, y, img.At(crop.Min.X+x, crop.Min.Y+y))
		}
	}
	return out, nil
}

func findBestWindowForPID(pid uint32) (windows.Handle, error) {
	var best windows.Handle
	var bestArea int32

	callback := windows.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		var windowPID uint32
		procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
		if windowPID != pid {
			return 1
		}
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}
		iconic, _, _ := procIsIconic.Call(hwnd)
		if iconic != 0 {
			return 1
		}
		var rect windowRect
		if ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect))); ret == 0 {
			return 1
		}
		width := rect.Right - rect.Left
		height := rect.Bottom - rect.Top
		if width <= 0 || height <= 0 {
			return 1
		}
		area := width * height
		if area > bestArea {
			bestArea = area
			best = windows.Handle(hwnd)
		}
		return 1
	})

	procEnumWindows.Call(callback, 0)
	if best == 0 {
		return 0, errors.New("window not found")
	}
	return best, nil
}
